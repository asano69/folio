package store

import (
	"database/sql"
	"errors"
	"folio/internal/storage"
)

type Book struct {
	ID     string
	Title  string
	Source string
}

type Page struct {
	ID       int
	BookID   string
	Number   int
	Filename string
}

// UpsertBook inserts a new book or updates its title and source if it already exists.
func (s *Store) UpsertBook(b storage.Book) error {
	_, err := s.db.Exec(`
		INSERT INTO books (id, title, source)
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title  = excluded.title,
			source = excluded.source
	`, b.ID, b.Title, b.Source)
	return err
}

// UpsertPages replaces all pages for a book.
func (s *Store) UpsertPages(bookID string, pages []storage.Page) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM pages WHERE book_id = ?`, bookID); err != nil {
		return err
	}

	for _, p := range pages {
		if _, err := tx.Exec(`
			INSERT INTO pages (book_id, number, filename)
			VALUES (?, ?, ?)
		`, bookID, p.Number, p.Filename); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListBooks returns all books ordered by title.
func (s *Store) ListBooks() ([]Book, error) {
	rows, err := s.db.Query(`SELECT id, title, source FROM books ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Source); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// GetBook returns a single book by ID.
func (s *Store) GetBook(id string) (*Book, error) {
	var b Book
	err := s.db.QueryRow(`SELECT id, title, source FROM books WHERE id = ?`, id).
		Scan(&b.ID, &b.Title, &b.Source)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &b, err
}

// ListPages returns all pages for a book ordered by number.
func (s *Store) ListPages(bookID string) ([]Page, error) {
	rows, err := s.db.Query(`
		SELECT id, book_id, number, filename
		FROM pages
		WHERE book_id = ?
		ORDER BY number
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.BookID, &p.Number, &p.Filename); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}
