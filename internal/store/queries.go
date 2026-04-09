package store

import (
	"database/sql"
	"errors"

	"folio/internal/storage"
)

// Page attribute constants. Stored as plain strings with no DB CHECK constraint
// so the list can evolve without a schema migration.
const (
	AttrCover   = "cover"
	AttrTOC     = "toc"
	AttrSection = "section"
	AttrPage    = "page"
	AttrIndex   = "index"
	AttrOther   = "other"
)

// AttributeOption pairs a stored value with a human-readable label for the UI.
type AttributeOption struct {
	Value string
	Label string
}

// AllAttributeOptions lists every valid attribute in display order.
var AllAttributeOptions = []AttributeOption{
	{AttrCover, "Cover"},
	{AttrTOC, "Table of Contents"},
	{AttrSection, "Section"},
	{AttrPage, "Page"},
	{AttrIndex, "Index"},
	{AttrOther, "Other"},
}

// Book is the DB representation of a book.
type Book struct {
	ID     string
	Title  string
	Source string
}

// Page is the DB representation of a page.
type Page struct {
	ID       int
	BookID   string
	Number   int
	Filename string
	Hash     string
}

// Note holds user-authored metadata for a single page.
// PageHash is the SHA-256 of the page's uncompressed image bytes, which
// remains stable across re-scans and CBZ page deletions.
type Note struct {
	BookID    string
	PageHash  string
	Title     string
	Attribute string
	Body      string
	UpdatedAt string
}

// TocEntry is a single entry in the table of contents derived from section-attributed pages.
type TocEntry struct {
	PageNum int
	Title   string
}

// GetTOC returns all section-attributed pages for a book, ordered by page number.
// Pages without a title are included; the caller is responsible for fallback display.
func (s *Store) GetTOC(bookID string) ([]TocEntry, error) {
	rows, err := s.db.Query(`
		SELECT p.number, n.title
		FROM pages p
		JOIN notes n ON n.book_id = p.book_id AND n.page_hash = p.hash
		WHERE p.book_id = ? AND n.attribute = 'section'
		ORDER BY p.number
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []TocEntry
	for rows.Next() {
		var e TocEntry
		if err := rows.Scan(&e.PageNum, &e.Title); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
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
			INSERT INTO pages (book_id, number, filename, hash)
			VALUES (?, ?, ?, ?)
		`, bookID, p.Number, p.Filename, p.Hash); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpsertThumbnail inserts or replaces a thumbnail for a book.
func (s *Store) UpsertThumbnail(bookID string, data []byte) error {
	_, err := s.db.Exec(`
		INSERT INTO thumbnails (book_id, data)
		VALUES (?, ?)
		ON CONFLICT(book_id) DO UPDATE SET
			data       = excluded.data,
			created_at = CURRENT_TIMESTAMP
	`, bookID, data)
	return err
}

// GetThumbnail returns the JPEG thumbnail bytes for a book, or nil if not found.
func (s *Store) GetThumbnail(bookID string) ([]byte, error) {
	var data []byte
	err := s.db.QueryRow(`SELECT data FROM thumbnails WHERE book_id = ?`, bookID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return data, err
}

// HasThumbnail reports whether a thumbnail exists for the given book.
func (s *Store) HasThumbnail(bookID string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM thumbnails WHERE book_id = ?`, bookID).Scan(&count)
	return count > 0, err
}

// UpdateBookTitle updates the title of an existing book.
func (s *Store) UpdateBookTitle(id, title string) error {
	_, err := s.db.Exec(`UPDATE books SET title = ? WHERE id = ?`, title, id)
	return err
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

// GetBook returns a single book by ID, or nil if not found.
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
		SELECT id, book_id, number, filename, hash
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
		if err := rows.Scan(&p.ID, &p.BookID, &p.Number, &p.Filename, &p.Hash); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

// GetCoverPage returns the first page of a book, or nil if none exists.
func (s *Store) GetCoverPage(bookID string) (*Page, error) {
	var p Page
	err := s.db.QueryRow(`
		SELECT id, book_id, number, filename, hash
		FROM pages
		WHERE book_id = ?
		ORDER BY number
		LIMIT 1
	`, bookID).Scan(&p.ID, &p.BookID, &p.Number, &p.Filename, &p.Hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// GetNote returns the note for a page identified by its hash, or a zero-value
// Note if none exists.
func (s *Store) GetNote(bookID, pageHash string) (Note, error) {
	var n Note
	err := s.db.QueryRow(`
		SELECT book_id, page_hash, title, attribute, body, updated_at
		FROM notes
		WHERE book_id = ? AND page_hash = ?
	`, bookID, pageHash).Scan(&n.BookID, &n.PageHash, &n.Title, &n.Attribute, &n.Body, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Note{BookID: bookID, PageHash: pageHash}, nil
	}
	return n, err
}

// UpsertNote inserts or updates the note for a page.
func (s *Store) UpsertNote(n Note) error {
	_, err := s.db.Exec(`
		INSERT INTO notes (book_id, page_hash, title, attribute, body, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(book_id, page_hash) DO UPDATE SET
			title      = excluded.title,
			attribute  = excluded.attribute,
			body       = excluded.body,
			updated_at = CURRENT_TIMESTAMP
	`, n.BookID, n.PageHash, n.Title, n.Attribute, n.Body)
	return err
}

// HasPages reports whether any pages are registered for the given book.
func (s *Store) HasPages(bookID string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pages WHERE book_id = ?`, bookID).Scan(&count)
	return count > 0, err
}
