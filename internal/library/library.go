package library

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Library struct {
	Path  string
	Books []Book
}

func NewLibrary(path string) (*Library, error) {
	lib := &Library{
		Path:  path,
		Books: []Book{},
	}

	if err := lib.Scan(); err != nil {
		return nil, err
	}

	return lib, nil
}

func (lib *Library) Scan() error {
	entries, err := os.ReadDir(lib.Path)
	if err != nil {
		return err
	}

	books := []Book{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		book, err := lib.scanBook(entry.Name())
		if err != nil {
			continue // スキップ
		}

		if len(book.Pages) > 0 {
			books = append(books, book)
		}
	}

	// 本をIDでソート
	sort.Slice(books, func(i, j int) bool {
		return books[i].ID < books[j].ID
	})

	lib.Books = books
	return nil
}

func (lib *Library) scanBook(dirName string) (Book, error) {
	bookPath := filepath.Join(lib.Path, dirName)
	entries, err := os.ReadDir(bookPath)
	if err != nil {
		return Book{}, err
	}

	pages := []Page{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		ext := strings.ToLower(filepath.Ext(filename))

		// 画像ファイルのみ
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}

		page := Page{
			Filename: filename,
			Path:     filepath.Join(dirName, filename),
		}
		pages = append(pages, page)
	}

	// ファイル名でソート
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Filename < pages[j].Filename
	})

	// ページ番号を割り当て
	for i := range pages {
		pages[i].Number = i + 1
	}

	return Book{
		ID:    dirName,
		Title: dirName, // 現時点ではディレクトリ名をタイトルとして使用
		Pages: pages,
	}, nil
}

func (lib *Library) GetBook(id string) *Book {
	for i := range lib.Books {
		if lib.Books[i].ID == id {
			return &lib.Books[i]
		}
	}
	return nil
}
