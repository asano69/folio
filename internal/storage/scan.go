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

// Scan walks the library root recursively and opens every .cbz file found.
// CBZ files are processed concurrently using a worker pool sized to GOMAXPROCS.
// It returns all successfully scanned books. Errors for individual files are
// printed to stderr and skipped so a single bad file does not abort the scan.
func Scan(libraryPath string) ([]Book, error) {
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
				book, err := openCBZ(path)
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
