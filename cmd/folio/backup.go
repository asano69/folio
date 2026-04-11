package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"folio/internal/config"
)

// runBackup copies the SQLite database file to the backup directory.
// The destination filename includes a timestamp so each backup is distinct.
// SQLite's WAL mode means the file may be accompanied by -shm and -wal
// sidecar files during active writes; copying the main .db file alone is
// safe when no write transaction is in progress (which is the case here
// because we open no DB connection before copying).
func runBackup(cfg *config.Config) error {
	srcPath := filepath.Join(cfg.DataPath, "folio.db")
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("database not found at %s: %w", srcPath, err)
	}

	if err := os.MkdirAll(cfg.BackupPath, 0755); err != nil {
		return fmt.Errorf("create backup directory %s: %w", cfg.BackupPath, err)
	}

	timestamp := time.Now().Format("20060102-150405")
	destFilename := fmt.Sprintf("folio-%s.db", timestamp)
	destPath := filepath.Join(cfg.BackupPath, destFilename)

	if err := copyFile(srcPath, destPath); err != nil {
		return fmt.Errorf("copy database to %s: %w", destPath, err)
	}

	fmt.Printf("Backup written to %s\n", destPath)
	return nil
}

// runRestore replaces the active database file with the specified backup file.
// The current database is overwritten; there is no automatic pre-restore backup.
// The caller should ensure the server is not running before restoring.
func runRestore(cfg *config.Config, backupPath string) error {
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found at %s: %w", backupPath, err)
	}

	if err := os.MkdirAll(cfg.DataPath, 0755); err != nil {
		return fmt.Errorf("create data directory %s: %w", cfg.DataPath, err)
	}

	destPath := filepath.Join(cfg.DataPath, "folio.db")

	if err := copyFile(backupPath, destPath); err != nil {
		return fmt.Errorf("restore database from %s: %w", backupPath, err)
	}

	fmt.Printf("Database restored from %s to %s\n", backupPath, destPath)
	return nil
}

// copyFile copies the file at src to dst, creating or truncating dst.
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
