package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"folio/internal/config"
	"folio/internal/storage"
	"folio/internal/store"
)

func runScan(cfg *config.Config, scanPath string) error {
	db, err := store.Open(cfg.DataPath)
	if err != nil {
		return err
	}
	defer db.Close()

	absScanPath, err := filepath.Abs(scanPath)
	if err != nil {
		return fmt.Errorf("resolve scan path: %w", err)
	}

	// Restrict missing-book detection to books under the scanned directory.
	allBooks, err := db.ListBooksUnderPath(absScanPath)
	if err != nil {
		return fmt.Errorf("list books: %w", err)
	}

	fmt.Printf("Scanning %s\n", absScanPath)

	// Phase 1: lightweight meta scan — reads only folio.json from each CBZ.
	metaBooks, err := storage.ScanMeta(absScanPath)
	if err != nil {
		return err
	}

	// Classify books: unchanged (mtime matches DB) vs. need full open.
	var unchangedBooks []storage.Book
	var fullOpenPaths []string

	for _, b := range metaBooks {
		if b.ID == "" {
			fullOpenPaths = append(fullOpenPaths, b.Source)
			continue
		}
		existing, err := db.GetBook(b.ID)
		if err != nil {
			return err
		}
		if existing == nil || existing.FileMtime != b.FileMtime {
			fullOpenPaths = append(fullOpenPaths, b.Source)
		} else {
			unchangedBooks = append(unchangedBooks, b)
		}
	}

	foundIDs := make(map[string]struct{}, len(unchangedBooks)+len(fullOpenPaths))

	// Upsert unchanged books to keep source path current and clear missing_since.
	for _, b := range unchangedBooks {
		foundIDs[b.ID] = struct{}{}
		if err := db.UpsertBook(b); err != nil {
			return fmt.Errorf("upsert book %s: %w", b.ID, err)
		}
	}

	// Phase 2: full open (with hash computation) for new/changed books.
	// Workers run in parallel bounded by GOMAXPROCS; each result is committed
	// to the DB immediately so books become queryable as soon as they finish.
	type openResult struct {
		book storage.Book
		path string
		err  error
	}

	jobCh := make(chan string, len(fullOpenPaths))
	for _, p := range fullOpenPaths {
		jobCh <- p
	}
	close(jobCh)

	resultCh := make(chan openResult, runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobCh {
				book, err := storage.OpenBook(path)
				resultCh <- openResult{book, path, err}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Commit each result to the DB as it arrives.
	// UpsertPages uses a merge algorithm to preserve stable page IDs:
	// existing notes, drawings, and status records are not affected.
	var changedBooks []storage.Book
	for r := range resultCh {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "scan: skip %s: %v\n", r.path, r.err)
			continue
		}
		foundIDs[r.book.ID] = struct{}{}
		if err := db.UpsertBook(r.book); err != nil {
			return fmt.Errorf("upsert book %s: %w", r.book.ID, err)
		}
		if err := db.UpsertPages(r.book.ID, r.book.Pages); err != nil {
			return fmt.Errorf("upsert pages %s: %w", r.book.ID, err)
		}
		fmt.Printf("  %s (%d pages)\n", r.book.Title, len(r.book.Pages))
		changedBooks = append(changedBooks, r.book)
	}

	allFound := make([]storage.Book, 0, len(unchangedBooks)+len(changedBooks))
	allFound = append(allFound, unchangedBooks...)
	allFound = append(allFound, changedBooks...)

	if err := generateMissingBookThumbnails(cfg.CachePath, allFound); err != nil {
		return err
	}

	// Mark books that were in the DB but not found on disk.
	var missingCount int
	for _, b := range allBooks {
		if _, found := foundIDs[b.ID]; !found {
			missingCount++
			if err := db.MarkBookMissing(b.ID); err != nil {
				return fmt.Errorf("mark missing %s: %w", b.ID, err)
			}
		}
	}

	fmt.Printf("Done. %d books found (%d updated), %d missing.\n",
		len(allFound), len(changedBooks), missingCount)
	return nil
}

// generateMissingBookThumbnails generates and stores book-level thumbnails for
// any book that does not yet have a cached thumbnail file. Generation is
// parallelised via runWorkerPool; file writes are sequential.
func generateMissingBookThumbnails(cachePath string, books []storage.Book) error {
	type thumbJob struct {
		bookID string
		source string
		title  string
	}
	type thumbResult struct {
		bookID string
		title  string
		data   []byte
		err    error
	}

	var jobs []thumbJob
	for _, b := range books {
		if !storage.BookThumbnailExists(cachePath, b.ID) {
			jobs = append(jobs, thumbJob{b.ID, b.Source, b.Title})
		}
	}

	if len(jobs) == 0 {
		return nil
	}

	results := runWorkerPool(jobs, func(j thumbJob) thumbResult {
		data, err := storage.GenerateBookThumbnail(j.source)
		return thumbResult{j.bookID, j.title, data, err}
	})

	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "  thumbnail skip %s: %v\n", r.title, r.err)
			continue
		}
		if err := storage.WriteBookThumbnail(cachePath, r.bookID, r.data); err != nil {
			return fmt.Errorf("write thumbnail %s: %w", r.bookID, err)
		}
	}
	return nil
}
