package main

import (
	"fmt"
	"os"

	"folio/internal/config"
	"folio/internal/storage"
	"folio/internal/store"
)

// runThumbnail regenerates the book-level thumbnail for a single book
// identified by UUID.
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

// runImageThumbnails generates page-level thumbnails for one book (when bookID
// is non-empty) or for all non-missing books. Images that already have a
// thumbnail in page_thumbnails are skipped.
func runImageThumbnails(cfg *config.Config, bookID string) error {
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
		bookID   string
		source   string
		reqCount int
		reqs     []storage.ImageThumbnailRequest
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
		images, err := db.ListImages(b.ID)
		if err != nil {
			return fmt.Errorf("list images %s: %w", b.ID, err)
		}
		var reqs []storage.ImageThumbnailRequest
		for _, img := range images {
			if img.Hash == "" {
				fmt.Fprintf(os.Stderr, "  skip image %d of %s: no hash (run folio hash <uuid>)\n", img.Number, b.ID)
				continue
			}
			exists, err := db.HasImageThumbnail(b.ID, img.Hash)
			if err != nil {
				return fmt.Errorf("check image thumbnail %s/%s: %w", b.ID, img.Hash, err)
			}
			if !exists {
				reqs = append(reqs, storage.ImageThumbnailRequest{Filename: img.Filename, Hash: img.Hash})
			}
		}
		if len(reqs) > 0 {
			jobs = append(jobs, bookJob{b.ID, b.Source, len(reqs), reqs})
		}
	}

	if len(jobs) == 0 {
		fmt.Println("All image thumbnails are up to date.")
		return nil
	}

	total := 0
	for _, j := range jobs {
		total += j.reqCount
	}
	fmt.Printf("Generating %d image thumbnails across %d books...\n", total, len(jobs))

	// Each worker opens a CBZ once and processes all queued images in a single
	// pass, amortising the cost of reading the ZIP central directory.
	results := runWorkerPool(jobs, func(j bookJob) bookResult {
		res, err := storage.GenerateImageThumbnails(j.source, j.reqs)
		return bookResult{j.bookID, j.reqCount, res, err}
	})

	var done, skipped int
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "  skip book %s: %v\n", r.bookID, r.err)
			skipped += r.reqCount
			continue
		}
		for _, it := range r.results {
			if err := db.UpsertImageThumbnail(r.bookID, it.Hash, it.Data); err != nil {
				return fmt.Errorf("store image thumbnail: %w", err)
			}
			done++
		}
	}

	fmt.Printf("Done. %d generated, %d skipped.\n", done, skipped)
	return nil
}
