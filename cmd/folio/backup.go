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
//
// The server must be stopped before running this command. Restoring while the
// server is running risks data corruption because SQLite connections hold file
// locks and may be mid-transaction when the file is replaced.
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

	if err := copyFile(absBackupPath, destPath); err != nil {
		return fmt.Errorf("restore database: %w", err)
	}

	fmt.Printf("Database restored from %s to %s\n", absBackupPath, destPath)
	return nil
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
