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
		if err := runImageThumbnails(cfg, bookID); err != nil {
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

	// Phase 1: lightweight meta scan — reads only folio.json from each CBZ.
	// This gives us each book's UUID and file mtime without hashing any images.
	metaBooks, err := storage.ScanMeta(absScanPath)
	if err != nil {
		return err
	}

	// Classify books: unchanged (mtime matches DB) vs. need full open.
	// A book needs a full open when:
	//   - It has no folio.json yet (ID is empty), or
	//   - It is not yet in the DB, or
	//   - Its file mtime differs from the stored value (contents changed).
	var unchangedBooks []storage.Book
	var fullOpenPaths []string

	for _, b := range metaBooks {
		if b.ID == "" {
			// No folio.json yet; full open will generate one.
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

	// Phase 2: full open (with hash computation) only for new/changed books.
	// Parallelised across GOMAXPROCS workers, same pattern as storage.Scan.
	var changedBooks []storage.Book
	if len(fullOpenPaths) > 0 {
		type result struct {
			book storage.Book
			path string
			err  error
		}

		jobs := make(chan string, len(fullOpenPaths))
		for _, p := range fullOpenPaths {
			jobs <- p
		}
		close(jobs)

		results := make(chan result, len(fullOpenPaths))

		var wg sync.WaitGroup
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for path := range jobs {
					book, err := storage.OpenBook(path)
					results <- result{book, path, err}
				}
			}()
		}
		go func() {
			wg.Wait()
			close(results)
		}()

		for r := range results {
			if r.err != nil {
				fmt.Fprintf(os.Stderr, "scan: skip %s: %v\n", r.path, r.err)
				continue
			}
			changedBooks = append(changedBooks, r.book)
		}
	}

	foundIDs := make(map[string]struct{}, len(unchangedBooks)+len(changedBooks))

	// Upsert unchanged books to keep source path current (the file may have
	// moved to a different subdirectory) and clear missing_since if set.
	for _, b := range unchangedBooks {
		foundIDs[b.ID] = struct{}{}
		if err := db.UpsertBook(b); err != nil {
			return fmt.Errorf("upsert book %s: %w", b.ID, err)
		}
	}

	// Upsert new/changed books and always refresh their image records, since the
	// CBZ contents may have changed (images added, removed, or reordered).
	for _, b := range changedBooks {
		foundIDs[b.ID] = struct{}{}
		if err := db.UpsertBook(b); err != nil {
			return fmt.Errorf("upsert book %s: %w", b.ID, err)
		}
		if err := db.UpsertImages(b.ID, b.Pages); err != nil {
			return fmt.Errorf("upsert images %s: %w", b.ID, err)
		}
		fmt.Printf("  %s (%d images)\n", b.Title, len(b.Pages))
	}

	// Combine all found books for the thumbnail phase.
	allFound := make([]storage.Book, 0, len(unchangedBooks)+len(changedBooks))
	allFound = append(allFound, unchangedBooks...)
	allFound = append(allFound, changedBooks...)

	// Generate missing book-level thumbnails concurrently, then write to DB.
	// Thumbnail generation (image decode + resize) is CPU-bound and safe to
	// parallelise because each goroutine reads from a different CBZ file.
	// DB writes remain sequential to stay within SQLite's single-writer model.
	type thumbJob struct {
		bookID string
		source string
		title  string
	}
	var thumbJobs []thumbJob
	for _, b := range allFound {
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

	fmt.Printf("Done. %d books found (%d updated), %d missing.\n",
		len(allFound), len(changedBooks), missingCount)
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

// runImageThumbnails generates image-level thumbnails for a single book (when
// bookID is non-empty) or for all non-missing books in the library. Images that
// already have a thumbnail in page_thumbnails are skipped.
//
// Jobs are batched per book: each worker opens a CBZ once and processes all
// images that need thumbnails in a single pass, avoiding repeated ZIP central
// directory reads. DB writes remain sequential.
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

	// Collect one job per book containing only images that need thumbnails.
	type bookJob struct {
		bookID   string
		source   string
		reqCount int
		reqs     []storage.ImageThumbnailRequest
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

	type bookResult struct {
		bookID   string
		reqCount int
		results  []storage.ImageThumbnailResult
		err      error
	}

	jobCh := make(chan bookJob, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	resultCh := make(chan bookResult, len(jobs))

	var wg sync.WaitGroup
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				res, err := storage.GenerateImageThumbnails(j.source, j.reqs)
				resultCh <- bookResult{j.bookID, j.reqCount, res, err}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var done, skipped int
	for r := range resultCh {
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

// runHash recomputes image hashes for a single book identified by UUID and
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

	if err := db.UpsertImages(bookID, b.Pages); err != nil {
		return err
	}

	fmt.Printf("Images updated for %s (%s): %d images\n", book.Title, bookID, len(b.Pages))
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: folio [server|scan [path]|thumbnail <uuid>|page-thumbnails [uuid]|hash <uuid>]\n")
}
