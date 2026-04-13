package main

import (
	"fmt"
	"os"

	"folio/internal/config"
	"folio/internal/storage"
	"folio/internal/store"
)

// runBookThumbnail regenerates the book-level thumbnail for a single book
// identified by UUID.
func runBookThumbnail(cfg *config.Config, bookID string) error {
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

	data, err := storage.GenerateBookThumbnail(book.Source)
	if err != nil {
		return err
	}

	if err := storage.WriteBookThumbnail(cfg.CachePath, bookID, data); err != nil {
		return err
	}

	fmt.Printf("Thumbnail updated for %s (%s)\n", book.Title, bookID)
	return nil
}

// runPageThumbnails generates page-level thumbnails for one book (when bookID
// is non-empty) or for all non-missing books. Pages that already have a
// cached thumbnail file are skipped.
func runPageThumbnails(cfg *config.Config, bookID string) error {
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

	type bookJob struct {
		bookID string
		source string
		reqs   []storage.ImageThumbnailRequest
	}
	type bookResult struct {
		bookID   string
		reqCount int
		results  []storage.ImageThumbnailResult
		err      error
	}

	var jobs []bookJob
	for _, b := range books {
		if b.MissingSince != nil {
			continue // CBZ file is gone; nothing to read
		}
		pages, err := db.ListPages(b.ID)
		if err != nil {
			return fmt.Errorf("list pages %s: %w", b.ID, err)
		}
		var reqs []storage.ImageThumbnailRequest
		for _, p := range pages {
			if !storage.PageThumbnailExists(cfg.CachePath, b.ID, p.ID) {
				reqs = append(reqs, storage.ImageThumbnailRequest{
					PageID:   p.ID,
					Filename: p.Filename,
				})
			}
		}
		if len(reqs) > 0 {
			jobs = append(jobs, bookJob{b.ID, b.Source, reqs})
		}
	}

	if len(jobs) == 0 {
		fmt.Println("All page thumbnails are up to date.")
		return nil
	}

	total := 0
	for _, j := range jobs {
		total += len(j.reqs)
	}
	fmt.Printf("Generating %d page thumbnails across %d books...\n", total, len(jobs))

	// Each worker opens a CBZ once and processes all queued pages in a single
	// pass, amortising the cost of reading the ZIP central directory.
	results := runWorkerPool(jobs, func(j bookJob) bookResult {
		res, err := storage.GeneratePageThumbnails(j.source, j.reqs)
		return bookResult{j.bookID, len(j.reqs), res, err}
	})

	var done, skipped int
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "  skip book %s: %v\n", r.bookID, r.err)
			skipped += r.reqCount
			continue
		}
		for _, it := range r.results {
			if err := storage.WritePageThumbnail(cfg.CachePath, r.bookID, it.PageID, it.Data); err != nil {
				return fmt.Errorf("write page thumbnail: %w", err)
			}
			done++
		}
	}

	fmt.Printf("Done. %d generated, %d skipped.\n", done, skipped)
	return nil
}
