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
// MissingSince is non-nil when the CBZ file was not found during the last scan.
type Book struct {
	ID           string
	Title        string
	Source       string
	MissingSince *string
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

// UpsertBook inserts a new book or updates its title, source, and clears
// missing_since if it already exists. Clearing missing_since on upsert
// marks the book as present again after it reappears on disk.
func (s *Store) UpsertBook(b storage.Book) error {
	_, err := s.db.Exec(`
		INSERT INTO books (id, title, source)
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title         = excluded.title,
			source        = excluded.source,
			missing_since = NULL
	`, b.ID, b.Title, b.Source)
	return err
}

// MarkBookMissing sets missing_since to the current time for a book whose
// CBZ file was not found during a scan. It is a no-op if missing_since is
// already set, so the original disappearance time is preserved across scans.
func (s *Store) MarkBookMissing(id string) error {
	_, err := s.db.Exec(`
		UPDATE books SET missing_since = CURRENT_TIMESTAMP
		WHERE id = ? AND missing_since IS NULL
	`, id)
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

// ListBooks returns all books ordered by title, including missing ones.
func (s *Store) ListBooks() ([]Book, error) {
	rows, err := s.db.Query(`SELECT id, title, source, missing_since FROM books ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Source, &b.MissingSince); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// ListBooksUnderPath returns books whose source path is under the given
// directory. Used by partial scans to restrict missing-book detection to
// the scanned subtree only.
func (s *Store) ListBooksUnderPath(dirPath string) ([]Book, error) {
	// Match source paths that start with dirPath followed by a separator,
	// e.g. "/library/manga/book.cbz" matches dirPath="/library/manga".
	rows, err := s.db.Query(
		`SELECT id, title, source, missing_since FROM books WHERE source LIKE ? || '/%'`,
		dirPath,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Source, &b.MissingSince); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// GetBook returns a single book by ID, or nil if not found.
func (s *Store) GetBook(id string) (*Book, error) {
	var b Book
	err := s.db.QueryRow(`SELECT id, title, source, missing_since FROM books WHERE id = ?`, id).
		Scan(&b.ID, &b.Title, &b.Source, &b.MissingSince)
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

// Collection is the DB representation of a user-defined book list.
type Collection struct {
	ID        int
	Title     string
	BookCount int
}

// ListCollections returns all collections ordered by title, with book counts.
func (s *Store) ListCollections() ([]Collection, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.title, COUNT(cb.book_id)
		FROM collections c
		LEFT JOIN collection_books cb ON cb.collection_id = c.id
		GROUP BY c.id
		ORDER BY c.title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []Collection
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.ID, &c.Title, &c.BookCount); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// CreateCollection inserts a new collection and returns its ID.
func (s *Store) CreateCollection(title string) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO collections (title) VALUES (?)`, title)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// RenameCollection updates the title of an existing collection.
func (s *Store) RenameCollection(id int, title string) error {
	_, err := s.db.Exec(`UPDATE collections SET title = ? WHERE id = ?`, title, id)
	return err
}

// DeleteCollection removes a collection and all its book memberships.
func (s *Store) DeleteCollection(id int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM collection_books WHERE collection_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM collections WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// AddBookToCollection adds a book to a collection. Duplicate inserts are ignored.
func (s *Store) AddBookToCollection(collectionID int, bookID string) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO collection_books (collection_id, book_id) VALUES (?, ?)
	`, collectionID, bookID)
	return err
}

// RemoveBookFromCollection removes a book from a collection.
func (s *Store) RemoveBookFromCollection(collectionID int, bookID string) error {
	_, err := s.db.Exec(`
		DELETE FROM collection_books WHERE collection_id = ? AND book_id = ?
	`, collectionID, bookID)
	return err
}

// ListBooksInCollection returns books belonging to a collection, ordered by title.
func (s *Store) ListBooksInCollection(collectionID int) ([]Book, error) {
	rows, err := s.db.Query(`
		SELECT b.id, b.title, b.source, b.missing_since
		FROM books b
		JOIN collection_books cb ON cb.book_id = b.id
		WHERE cb.collection_id = ?
		ORDER BY b.title
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Source, &b.MissingSince); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// CountAllBooks returns the number of non-missing books in the library.
func (s *Store) CountAllBooks() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM books WHERE missing_since IS NULL`).Scan(&n)
	return n, err
}
