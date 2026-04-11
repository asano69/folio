package main

import (
	"fmt"

	"folio/internal/config"
	"folio/internal/storage"
	"folio/internal/store"
)

// runHash recomputes image hashes for a single book identified by UUID and
// updates the DB. Use this after manually modifying a CBZ's image contents
// when file mtime alone is not sufficient to trigger a re-hash during folio scan.
func runHash(cfg *config.Config, bookID string) error {
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

	b, err := storage.OpenBook(book.Source)
	if err != nil {
		return err
	}

	if err := db.UpsertImages(bookID, b.Pages); err != nil {
		return err
	}

	fmt.Printf("Images updated for %s (%s): %d images\n", book.Title, bookID, len(b.Pages))
	return nil
}
