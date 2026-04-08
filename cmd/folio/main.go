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
	case "thumbnail":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: folio thumbnail <book-uuid>\n")
			os.Exit(1)
		}
		if err := runThumbnail(cfg, os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "thumbnail: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func runServer(cfg *config.Config) {
	srv, err := newServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
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

		// Generate thumbnail only when one does not yet exist.
		exists, err := db.HasThumbnail(b.ID)
		if err != nil {
			return fmt.Errorf("check thumbnail %s: %w", b.ID, err)
		}
		if !exists {
			data, err := storage.GenerateThumbnail(b.Source)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  thumbnail skip %s: %v\n", b.Title, err)
			} else if err := db.UpsertThumbnail(b.ID, data); err != nil {
				return fmt.Errorf("store thumbnail %s: %w", b.ID, err)
			}
		}

		fmt.Printf("  %s (%d pages)\n", b.Title, len(b.Pages))
	}

	fmt.Printf("Done. %d books registered.\n", len(books))
	return nil
}

// runThumbnail regenerates the thumbnail for a single book identified by UUID.
func runThumbnail(cfg *config.Config, bookID string) error {
	db, err := store.Open(cfg.DataPath)
	if err != nil {
		return err
	}
	defer db.Close()

	book, err := db.GetBook(bookID)
	if err != nil {
		return err
	}
	if book == nil {
		return fmt.Errorf("book %s not found", bookID)
	}

	data, err := storage.GenerateThumbnail(book.Source)
	if err != nil {
		return err
	}

	if err := db.UpsertThumbnail(bookID, data); err != nil {
		return err
	}

	fmt.Printf("Thumbnail updated for %s (%s)\n", book.Title, bookID)
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: folio [server|scan|thumbnail <uuid>]\n")
}
