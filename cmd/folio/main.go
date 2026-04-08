package main

import (
	"fmt"
	"os"

	"folio/internal/config"
	"folio/internal/storage"
	"folio/internal/store"
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
		if err := runScan(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "scan: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func runScan(cfg *config.Config) error {
	db, err := store.Open(cfg.DataPath)
	if err != nil {
		return err
	}
	defer db.Close()

	fmt.Printf("Scanning %s\n", cfg.LibraryPath)
	books, err := storage.Scan(cfg.LibraryPath)
	if err != nil {
		return err
	}

	for _, b := range books {
		if err := db.UpsertBook(b); err != nil {
			return fmt.Errorf("upsert book %s: %w", b.ID, err)
		}
		if err := db.UpsertPages(b.ID, b.Pages); err != nil {
			return fmt.Errorf("upsert pages %s: %w", b.ID, err)
		}
		fmt.Printf("  %s (%d pages)\n", b.Title, len(b.Pages))
	}

	fmt.Printf("Done. %d books registered.\n", len(books))
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: folio [server|scan]\n")
}
