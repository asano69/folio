package main

import (
	"fmt"
	"os"

	"folio/internal/config"
	"folio/internal/storage"
	"folio/internal/store"
)

// runInject writes book metadata from the DB back to folio.json inside each CBZ.
// Only books whose stored folio.json differs from the DB are rewritten.
// When bookID is non-empty, only that book is processed.
func runInject(cfg *config.Config, bookID string) error {
	db, err := store.Open(cfg.DataPath)
	if err != nil {
		return err
	}
	defer db.Close()

	var books []store.Book
	if bookID != "" {
		book, err := db.GetBook(bookID)
		if err != nil {
			return err
		}
		if book == nil {
			return fmt.Errorf("book %s not found", bookID)
		}
		books = []store.Book{*book}
	} else {
		books, err = db.ListBooks()
		if err != nil {
			return fmt.Errorf("list books: %w", err)
		}
	}

	var updated, upToDate, skipped int
	for _, b := range books {
		if b.MissingSince != nil {
			skipped++
			continue
		}
		changed, err := storage.InjectBookMeta(b.Source, storeBookToStorageBook(b))
		if err != nil {
			fmt.Fprintf(os.Stderr, "  inject: skip %s: %v\n", b.Title, err)
			skipped++
			continue
		}
		if changed {
			fmt.Printf("  updated: %s\n", b.Title)
			updated++
		} else {
			upToDate++
		}
	}

	fmt.Printf("Done. %d updated, %d up to date, %d skipped.\n", updated, upToDate, skipped)
	return nil
}

// storeBookToStorageBook converts a store.Book to a storage.Book for use with
// storage-layer functions. Only metadata fields are copied; source and mtime
// are not relevant for injection.
func storeBookToStorageBook(b store.Book) storage.Book {
	return storage.Book{
		ID:           b.ID,
		Title:        b.Title,
		Type:         b.Type,
		Abstract:     b.Abstract,
		Language:     b.Language,
		Author:       b.Author,
		Translator:   b.Translator,
		OrigTitle:    b.OrigTitle,
		Edition:      b.Edition,
		Volume:       b.Volume,
		Series:       b.Series,
		SeriesNumber: b.SeriesNumber,
		Publisher:    b.Publisher,
		Year:         b.Year,
		Note:         b.Note,
		Keywords:     b.Keywords,
		ISBN:         b.ISBN,
		Links:        b.Links,
	}
}
