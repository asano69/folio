package main

import (
	"fmt"
	"folio/internal/config"
	"folio/internal/storage"
	"folio/internal/store"
	"os"
	"path/filepath"
	"runtime"
	"sync"
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
		// Optional book UUID; omit to process the entire library.
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

func runScan(cfg *config.Config, scanPath string) error {
	db, err := store.Open(cfg.DataPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Resolve to absolute path so it matches the absolute paths stored in
	// the DB (storage.Scan always writes absolute paths).
	absScanPath, err := filepath.Abs(scanPath)
	if err != nil {
		return fmt.Errorf("resolve scan path: %w", err)
	}

	// Restrict missing-book detection to books under the scanned directory.
	// A partial scan must not mark books outside the scan path as missing.
	allBooks, err := db.ListBooksUnderPath(absScanPath)
	if err != nil {
		return fmt.Errorf("list books: %w", err)
	}

	fmt.Printf("Scanning %s\n", absScanPath)
	books, err := storage.Scan(absScanPath)

	if err != nil {
		return err
	}

	// Track which book IDs were found on disk in this scan.
	foundIDs := make(map[string]struct{}, len(books))

	// Phase 1: update book and page records (sequential DB writes).
	for _, b := range books {
		foundIDs[b.ID] = struct{}{}

		if err := db.UpsertBook(b); err != nil {
			return fmt.Errorf("upsert book %s: %w", b.ID, err)
		}

		// Skip page registration if already present. Use "folio hash <uuid>"
		// to force a recalculation when the CBZ contents have changed.
		hasPages, err := db.HasPages(b.ID)
		if err != nil {
			return fmt.Errorf("check pages %s: %w", b.ID, err)
		}
		if !hasPages {
			if err := db.UpsertPages(b.ID, b.Pages); err != nil {
				return fmt.Errorf("upsert pages %s: %w", b.ID, err)
			}
		}

		fmt.Printf("  %s (%d pages)\n", b.Title, len(b.Pages))
	}

	// Phase 2: generate missing thumbnails concurrently, then write to DB.
	// Thumbnail generation (image decode + resize) is CPU-bound and safe to
	// parallelise because each goroutine reads from a different CBZ file.
	// DB writes remain sequential to stay within SQLite's single-writer model.
	type thumbJob struct {
		bookID string
		source string
		title  string
	}
	var thumbJobs []thumbJob
	for _, b := range books {
		exists, err := db.HasThumbnail(b.ID)
		if err != nil {
			return fmt.Errorf("check thumbnail %s: %w", b.ID, err)
		}
		if !exists {
			thumbJobs = append(thumbJobs, thumbJob{b.ID, b.Source, b.Title})
		}
	}

	if len(thumbJobs) > 0 {
		type thumbResult struct {
			bookID string
			title  string
			data   []byte
			err    error
		}

		jobs := make(chan thumbJob, len(thumbJobs))
		for _, j := range thumbJobs {
			jobs <- j
		}
		close(jobs)

		results := make(chan thumbResult, len(thumbJobs))

		var wg sync.WaitGroup
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobs {
					data, err := storage.GenerateThumbnail(j.source)
					results <- thumbResult{bookID: j.bookID, title: j.title, data: data, err: err}
				}
			}()
		}
		go func() {
			wg.Wait()
			close(results)
		}()

		for r := range results {
			if r.err != nil {
				fmt.Fprintf(os.Stderr, "  thumbnail skip %s: %v\n", r.title, r.err)
				continue
			}
			if err := db.UpsertThumbnail(r.bookID, r.data); err != nil {
				return fmt.Errorf("store thumbnail %s: %w", r.bookID, err)
			}
		}
	}

	// Mark books that were in the DB but not found on disk.
	// MarkBookMissing only sets missing_since on books where it is still NULL,
	// preserving the original disappearance time across subsequent scans.
	var missingCount int
	for _, b := range allBooks {
		if _, found := foundIDs[b.ID]; !found {
			missingCount++
			if err := db.MarkBookMissing(b.ID); err != nil {
				return fmt.Errorf("mark missing %s: %w", b.ID, err)
			}
		}
	}

	fmt.Printf("Done. %d books found, %d missing.\n", len(books), missingCount)
	return nil
}

// runThumbnail regenerates the book-level thumbnail for a single book identified by UUID.
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

// runPageThumbnails generates page-level thumbnails for a single book (when
// bookID is non-empty) or for all non-missing books in the library. Pages that
// already have a thumbnail in page_thumbnails are skipped by default.
// Generation is parallelised across GOMAXPROCS workers; DB writes are sequential.
func runPageThumbnails(cfg *config.Config, bookID string) error {
	db, err := store.Open(cfg.DataPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Resolve the target book list.
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

	// Collect pages that are missing a thumbnail.
	type pageJob struct {
		bookID   string
		source   string
		pageHash string
		filename string
	}
	var jobs []pageJob

	for _, b := range books {
		if b.MissingSince != nil {
			continue // CBZ file is gone; nothing to read
		}
		pages, err := db.ListPages(b.ID)
		if err != nil {
			return fmt.Errorf("list pages %s: %w", b.ID, err)
		}
		for _, p := range pages {
			exists, err := db.HasPageThumbnail(b.ID, p.Hash)
			if err != nil {
				return fmt.Errorf("check page thumbnail %s/%s: %w", b.ID, p.Hash, err)
			}
			if !exists {
				jobs = append(jobs, pageJob{b.ID, b.Source, p.Hash, p.Filename})
			}
		}
	}

	if len(jobs) == 0 {
		fmt.Println("All page thumbnails are up to date.")
		return nil
	}

	fmt.Printf("Generating %d page thumbnails...\n", len(jobs))

	type thumbResult struct {
		bookID   string
		pageHash string
		data     []byte
		err      error
	}

	jobCh := make(chan pageJob, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	results := make(chan thumbResult, len(jobs))

	var wg sync.WaitGroup
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				data, err := storage.GeneratePageThumbnail(j.source, j.filename)
				results <- thumbResult{j.bookID, j.pageHash, data, err}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var done, skipped int
	for r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "  skip: %v\n", r.err)
			skipped++
			continue
		}
		if err := db.UpsertPageThumbnail(r.bookID, r.pageHash, r.data); err != nil {
			return fmt.Errorf("store page thumbnail: %w", err)
		}
		done++
	}

	fmt.Printf("Done. %d generated, %d skipped.\n", done, skipped)
	return nil
}

// runHash recomputes page hashes for a single book identified by UUID and
// updates the DB. Use this when the CBZ contents have changed since the last scan.
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

	if err := db.UpsertPages(bookID, b.Pages); err != nil {
		return err
	}

	fmt.Printf("Pages updated for %s (%s): %d pages\n", book.Title, bookID, len(b.Pages))
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: folio [server|scan [path]|thumbnail <uuid>|page-thumbnails [uuid]|hash <uuid>]\n")
}
