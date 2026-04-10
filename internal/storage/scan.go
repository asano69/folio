package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Scan walks the library root recursively and opens every .cbz file found,
// computing page hashes for all of them. Use ScanMeta when you only need book
// identity and modification time without the cost of hashing every page.
//
// CBZ files are processed concurrently using a worker pool sized to GOMAXPROCS.
// Errors for individual files are printed to stderr and skipped so a single
// bad file does not abort the scan.
func Scan(libraryPath string) ([]Book, error) {
	return scanWith(libraryPath, openCBZ)
}

// ScanMeta walks the library root recursively and reads only folio.json from
// each CBZ, skipping page listing and hash computation. It is much faster than
// Scan and is intended as the first phase of a two-phase scan: use ScanMeta to
// identify which books are new or changed, then call OpenBook only for those.
//
// Books whose CBZ does not yet contain folio.json are returned with an empty ID;
// the caller should follow up with OpenBook for those paths.
func ScanMeta(libraryPath string) ([]Book, error) {
	return scanWith(libraryPath, openCBZMeta)
}

// scanWith is the shared walk + worker-pool implementation used by both Scan
// and ScanMeta. The open function determines how much of each CBZ is read.
func scanWith(libraryPath string, open func(string) (Book, error)) ([]Book, error) {
	absPath, err := filepath.Abs(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve library path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("library path %q not accessible: %w", absPath, err)
	}

	var paths []string
	if err := filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.ToLower(filepath.Ext(path)) == ".cbz" {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if len(paths) == 0 {
		return nil, nil
	}

	jobs := make(chan string, len(paths))
	for _, p := range paths {
		jobs <- p
	}
	close(jobs)

	type result struct {
		book Book
		path string
		err  error
	}
	results := make(chan result, len(paths))

	var wg sync.WaitGroup
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				book, err := open(path)
				results <- result{book: book, path: path, err: err}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var books []Book
	for r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "scan: skip %s: %v\n", r.path, r.err)
			continue
		}
		books = append(books, r.book)
	}

	return books, nil
}
