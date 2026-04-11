package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"folio/internal/config"

	_ "modernc.org/sqlite"
)

// runBackup creates a consistent snapshot of the live database using VACUUM INTO.
// VACUUM INTO holds a read transaction for its duration, so it is safe to run
// while the server is active; ongoing writes are simply reflected or excluded
// depending on when they commit relative to the snapshot.
func runBackup(cfg *config.Config) error {
	srcPath := filepath.Join(cfg.DataPath, "folio.db")
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("database not found at %s: %w", srcPath, err)
	}

	if err := os.MkdirAll(cfg.BackupPath, 0755); err != nil {
		return fmt.Errorf("create backup directory %s: %w", cfg.BackupPath, err)
	}

	timestamp := time.Now().Format("20060102-150405")
	destPath := filepath.Join(cfg.BackupPath, fmt.Sprintf("folio-%s.db", timestamp))

	db, err := sql.Open("sqlite", srcPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// VACUUM INTO writes a compacted, consistent copy to destPath.
	// Safe to run against a WAL-mode database under concurrent read/write load.
	if _, err := db.Exec("VACUUM INTO ?", destPath); err != nil {
		return fmt.Errorf("vacuum into %s: %w", destPath, err)
	}

	fmt.Printf("Backup written to %s\n", destPath)
	return nil
}

// runRestore replaces the active database file with the specified backup file.
// The server must be stopped before running this command.
//
// Before copying, an exclusive lock is acquired on the live database to confirm
// no other process holds it open. If the lock cannot be acquired the server is
// likely still running and the command exits with an error.
//
// After copying, the WAL sidecar files (-shm, -wal) are removed so that SQLite
// opens the restored database in a clean state on next startup.
func runRestore(cfg *config.Config, backupPath string) error {
	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return fmt.Errorf("resolve backup path: %w", err)
	}
	if _, err := os.Stat(absBackupPath); err != nil {
		return fmt.Errorf("backup file not found at %s: %w", absBackupPath, err)
	}

	if err := os.MkdirAll(cfg.DataPath, 0755); err != nil {
		return fmt.Errorf("create data directory %s: %w", cfg.DataPath, err)
	}

	destPath := filepath.Join(cfg.DataPath, "folio.db")

	// Verify that no other process holds the database open by attempting to
	// acquire an exclusive lock. busy_timeout = 0 means fail immediately rather
	// than waiting, so a running server is detected at once.
	if err := checkDatabaseNotInUse(destPath); err != nil {
		return err
	}

	if err := copyFile(absBackupPath, destPath); err != nil {
		return fmt.Errorf("restore database: %w", err)
	}

	// Remove WAL sidecar files left over from the previous database session.
	// If these are not removed, SQLite may try to apply stale WAL entries to
	// the newly restored database on next open, corrupting it.
	removeWALSidecars(destPath)

	fmt.Printf("Database restored from %s to %s\n", absBackupPath, destPath)
	return nil
}

// checkDatabaseNotInUse opens the database with busy_timeout = 0 and attempts
// to acquire an exclusive lock. If another process holds the file open (e.g.
// the server is running), SQLite returns SQLITE_BUSY immediately and we surface
// a clear error to the user.
func checkDatabaseNotInUse(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// No existing database; nothing to lock-check.
		return nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open database for lock check: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA busy_timeout = 0"); err != nil {
		return fmt.Errorf("set busy_timeout: %w", err)
	}

	// BEGIN EXCLUSIVE fails immediately if another connection holds any lock.
	if _, err := db.Exec("BEGIN EXCLUSIVE"); err != nil {
		return fmt.Errorf("database is in use (is the server running?): %w", err)
	}
	db.Exec("ROLLBACK")

	return nil
}

// removeWALSidecars deletes the -shm and -wal files that accompany a WAL-mode
// SQLite database. Errors are ignored because the files may not exist.
func removeWALSidecars(dbPath string) {
	os.Remove(dbPath + "-shm")
	os.Remove(dbPath + "-wal")
}

// copyFile copies the file at src to dst, creating or truncating dst.
// Sync is called before closing so the data is flushed to disk.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}
