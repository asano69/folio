package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Scan walks the library root recursively and opens every .cbz file found.
// It returns all successfully scanned books. Errors for individual files are
// printed to stderr and skipped so a single bad file does not abort the scan.
func Scan(libraryPath string) ([]Book, error) {
	if _, err := os.Stat(libraryPath); err != nil {
		return nil, fmt.Errorf("library path %q not accessible: %w", libraryPath, err)
	}

	var books []Book

	err := filepath.WalkDir(libraryPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".cbz" {
			return nil
		}

		book, err := openCBZ(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan: skip %s: %v\n", path, err)
			return nil
		}

		books = append(books, book)
		return nil
	})

	return books, err
}
