package main

import (
	"fmt"
	"os"

	"folio/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cfg := config.Load()

	switch os.Args[1] {
	case "server":
		runServer(cfg)
	case "scan":
		scanPath := cfg.LibraryPath
		if len(os.Args) >= 3 {
			scanPath = os.Args[2]
		}
		if err := runScan(cfg, scanPath); err != nil {
			fmt.Fprintf(os.Stderr, "scan: %v\n", err)
			os.Exit(1)
		}
	case "thumbnail":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: folio thumbnail <book-uuid>\n")
			os.Exit(1)
		}
		if err := runThumbnail(cfg, os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "thumbnail: %v\n", err)
			os.Exit(1)
		}
	case "page-thumbnails":
		bookID := ""
		if len(os.Args) >= 3 {
			bookID = os.Args[2]
		}
		if err := runPageThumbnails(cfg, bookID); err != nil {
			fmt.Fprintf(os.Stderr, "page-thumbnails: %v\n", err)
			os.Exit(1)
		}
	case "hash":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: folio hash <book-uuid>\n")
			os.Exit(1)
		}
		if err := runHash(cfg, os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "hash: %v\n", err)
			os.Exit(1)
		}
	case "backup":
		if err := runBackup(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "backup: %v\n", err)
			os.Exit(1)
		}
	case "restore":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: folio restore <backup-file>\n")
			os.Exit(1)
		}
		if err := runRestore(cfg, os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "restore: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: folio <subcommand> [arguments]

Subcommands:
  server                     Start the HTTP server
  scan [path]                Scan CBZ files and sync the database
                             (default path: FOLIO_LIBRARY_PATH)
  thumbnail <uuid>           Regenerate the book-level thumbnail for one book
  page-thumbnails [uuid]     Generate page-level thumbnails (all books if omitted)
  hash <uuid>                Recompute image hashes for one book
  backup                     Copy the database to the backup directory
  restore <backup-file>      Replace the active database with a backup file

Environment:
  FOLIO_LIBRARY_PATH   CBZ library root        (default: ./library)
  FOLIO_DATA_PATH      SQLite database dir     (default: ./data)
  FOLIO_CACHE_PATH     Thumbnail cache dir     (default: ./cache)
  FOLIO_BACKUP_PATH    Backup output directory (default: ./backup)
  FOLIO_HOST           Server bind address     (default: 0.0.0.0)
  FOLIO_PORT           Server port             (default: 3000)

For full documentation see docs/cli.md.
`)
}
