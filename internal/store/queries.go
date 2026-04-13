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
	Status       string
	FileMtime    int64
	MissingSince *string
}

// Page is the DB representation of a single scanned image inside a CBZ.
// ID is stable across re-scans: UpsertPages merges by hash then by position
// rather than deleting and reinserting, so foreign keys from notes, page_status,
// page_drawings, etc. remain valid after a scan.
type Page struct {
	ID        int
	BookID    string
	Number    int
	Filename  string
	Hash      string
	Title     string
	Attribute string
}

// TocEntry is a single entry in the table of contents.
type TocEntry struct {
	PageNum int
	Title   string
}

// Collection is the DB representation of a user-defined book list.
type Collection struct {
	ID        int
	Title     string
	BookCount int
}

// ── Books ──────────────────────────────────────────────────────

// UpsertBook inserts a new book or updates its title, source, file_mtime, and
// clears missing_since. Status is excluded from the UPDATE so user-set values
// are preserved across re-scans.
func (s *Store) UpsertBook(b storage.Book) error {
	_, err := s.db.Exec(`
		INSERT INTO books (id, title, source, file_mtime)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title         = excluded.title,
			source        = excluded.source,
			file_mtime    = excluded.file_mtime,
			missing_since = NULL
	`, b.ID, b.Title, b.Source, b.FileMtime)
	return err
}

// MarkBookMissing sets missing_since to the current time for a book whose
// CBZ was not found during a scan. It is a no-op if missing_since is already
// set, preserving the original disappearance timestamp across repeated scans.
func (s *Store) MarkBookMissing(id string) error {
	_, err := s.db.Exec(`
		UPDATE books SET missing_since = CURRENT_TIMESTAMP
		WHERE id = ? AND missing_since IS NULL
	`, id)
	return err
}

// UpdateBookTitle updates the title of an existing book.
func (s *Store) UpdateBookTitle(id, title string) error {
	_, err := s.db.Exec(`UPDATE books SET title = ? WHERE id = ?`, title, id)
	return err
}

// ListBooks returns all books ordered by title, including missing ones.
func (s *Store) ListBooks() ([]Book, error) {
	rows, err := s.db.Query(`
		SELECT id, title, source, status, file_mtime, missing_since
		FROM books ORDER BY title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Source, &b.Status, &b.FileMtime, &b.MissingSince); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// ListBooksUnderPath returns books whose source path is under the given
// directory. Used by partial scans to restrict missing-book detection.
func (s *Store) ListBooksUnderPath(dirPath string) ([]Book, error) {
	rows, err := s.db.Query(
		`SELECT id, title, source, status, file_mtime, missing_since
		 FROM books WHERE source LIKE ? || '/%'`,
		dirPath,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Source, &b.Status, &b.FileMtime, &b.MissingSince); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// GetBook returns a single book by ID, or nil if not found.
func (s *Store) GetBook(id string) (*Book, error) {
	var b Book
	err := s.db.QueryRow(
		`SELECT id, title, source, status, file_mtime, missing_since FROM books WHERE id = ?`, id,
	).Scan(&b.ID, &b.Title, &b.Source, &b.Status, &b.FileMtime, &b.MissingSince)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &b, err
}

// CountAllBooks returns the number of non-missing books in the library.
func (s *Store) CountAllBooks() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM books WHERE missing_since IS NULL`).Scan(&n)
	return n, err
}

// ── Pages ──────────────────────────────────────────────────────

// UpsertPages merges the scanned image list into the pages table while
// preserving stable page IDs. The merge runs in two passes:
//
//  1. Match by hash (content identity): a page that moved to a different
//     position keeps its ID because its image content is unchanged.
//  2. Match remaining entries by position: a page whose content was replaced
//     in-place keeps its ID; only its hash (and filename) is updated.
//
// Pages with no match are inserted as new rows.
// Existing pages with no match are deleted; ON DELETE CASCADE propagates to
// notes, page_status, page_drawings, page_tags, and page_thumbnails.
//
// Because sections reference pages by the stable pages.id FK, no section
// rebuild is required after this operation.
func (s *Store) UpsertPages(bookID string, entries []storage.ImageEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Load existing pages for this book.
	rows, err := tx.Query(
		`SELECT id, number, hash FROM pages WHERE book_id = ? ORDER BY number`, bookID,
	)
	if err != nil {
		return err
	}
	type existingPage struct {
		id, number int
		hash       string
	}
	var existing []existingPage
	for rows.Next() {
		var p existingPage
		if err := rows.Scan(&p.id, &p.number, &p.hash); err != nil {
			rows.Close()
			return err
		}
		existing = append(existing, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	usedExisting := make([]bool, len(existing))
	usedNew := make([]bool, len(entries))
	// existingPage.id -> index into entries
	matches := make(map[int]int, len(existing))

	// Pass 1: match by hash (stable identity despite position change).
	for newIdx, entry := range entries {
		if entry.Hash == "" {
			continue
		}
		for exIdx, ex := range existing {
			if !usedExisting[exIdx] && ex.hash == entry.Hash {
				matches[ex.id] = newIdx
				usedExisting[exIdx] = true
				usedNew[newIdx] = true
				break
			}
		}
	}

	// Pass 2: match remaining entries by position (content replaced in-place).
	for newIdx, entry := range entries {
		if usedNew[newIdx] {
			continue
		}
		for exIdx, ex := range existing {
			if !usedExisting[exIdx] && ex.number == entry.Number {
				matches[ex.id] = newIdx
				usedExisting[exIdx] = true
				usedNew[newIdx] = true
				break
			}
		}
	}

	// Update matched pages: number, filename, or hash may have changed.
	for exID, newIdx := range matches {
		entry := entries[newIdx]
		if _, err := tx.Exec(
			`UPDATE pages SET number = ?, filename = ?, hash = ? WHERE id = ?`,
			entry.Number, entry.Filename, entry.Hash, exID,
		); err != nil {
			return err
		}
	}

	// Insert truly new pages (no existing page matched by hash or position).
	for newIdx, entry := range entries {
		if usedNew[newIdx] {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO pages (book_id, number, filename, hash) VALUES (?, ?, ?, ?)`,
			bookID, entry.Number, entry.Filename, entry.Hash,
		); err != nil {
			return err
		}
	}

	// Delete unmatched existing pages. ON DELETE CASCADE handles dependent rows.
	for exIdx, ex := range existing {
		if usedExisting[exIdx] {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM pages WHERE id = ?`, ex.id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListPages returns all pages for a book ordered by number.
func (s *Store) ListPages(bookID string) ([]Page, error) {
	rows, err := s.db.Query(`
		SELECT id, book_id, number, filename, hash, title, attribute
		FROM pages WHERE book_id = ? ORDER BY number
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.BookID, &p.Number, &p.Filename, &p.Hash, &p.Title, &p.Attribute); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

// GetPage returns a single page by its stable ID, or nil if not found.
func (s *Store) GetPage(pageID int) (*Page, error) {
	var p Page
	err := s.db.QueryRow(`
		SELECT id, book_id, number, filename, hash, title, attribute
		FROM pages WHERE id = ?
	`, pageID).Scan(&p.ID, &p.BookID, &p.Number, &p.Filename, &p.Hash, &p.Title, &p.Attribute)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// GetCoverPage returns the first page of a book, or nil if none exists.
func (s *Store) GetCoverPage(bookID string) (*Page, error) {
	var p Page
	err := s.db.QueryRow(`
		SELECT id, book_id, number, filename, hash, title, attribute
		FROM pages WHERE book_id = ? ORDER BY number LIMIT 1
	`, bookID).Scan(&p.ID, &p.BookID, &p.Number, &p.Filename, &p.Hash, &p.Title, &p.Attribute)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// HasPages reports whether any pages are registered for the given book.
func (s *Store) HasPages(bookID string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pages WHERE book_id = ?`, bookID).Scan(&count)
	return count > 0, err
}

// CountPages returns the total number of pages registered for a book.
func (s *Store) CountPages(bookID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pages WHERE book_id = ?`, bookID).Scan(&n)
	return n, err
}

// ── Page annotations (title, attribute) and notes (body) ──────

// UpsertPageEdit updates a page's title and attribute (stored on the pages row)
// and upserts the note body (stored in the unified notes table) in a single
// transaction. It also keeps the sections table in sync when the attribute
// changes to or from 'section'.
func (s *Store) UpsertPageEdit(pageID int, title, attribute, body string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Resolve book_id for the section sync call.
	var bookID string
	if err := tx.QueryRow(`SELECT book_id FROM pages WHERE id = ?`, pageID).Scan(&bookID); err != nil {
		return err
	}

	if _, err := tx.Exec(
		`UPDATE pages SET title = ?, attribute = ? WHERE id = ?`, title, attribute, pageID,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT INTO notes (page_id, body, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(page_id) DO UPDATE SET
			body       = excluded.body,
			updated_at = CURRENT_TIMESTAMP
	`, pageID, body); err != nil {
		return err
	}

	if err := syncSection(tx, pageID, bookID, attribute, title); err != nil {
		return err
	}

	return tx.Commit()
}

// GetPageNote returns the note body for a page, or an empty string if none exists.
func (s *Store) GetPageNote(pageID int) (string, error) {
	var body string
	err := s.db.QueryRow(`SELECT body FROM notes WHERE page_id = ?`, pageID).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return body, err
}

// ── Page drawings ──────────────────────────────────────────────

// GetPageDrawing returns the SVG markup for a page's drawing, or an empty
// string if no drawing has been saved.
func (s *Store) GetPageDrawing(pageID int) (string, error) {
	var svg string
	err := s.db.QueryRow(`SELECT svg FROM page_drawings WHERE page_id = ?`, pageID).Scan(&svg)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return svg, err
}

// UpsertPageDrawing saves or replaces an SVG drawing for a page.
// Passing nil removes any existing drawing.
func (s *Store) UpsertPageDrawing(pageID int, svg *string) error {
	if svg == nil {
		_, err := s.db.Exec(`DELETE FROM page_drawings WHERE page_id = ?`, pageID)
		return err
	}
	_, err := s.db.Exec(`
		INSERT INTO page_drawings (page_id, svg, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(page_id) DO UPDATE SET
			svg        = excluded.svg,
			updated_at = CURRENT_TIMESTAMP
	`, pageID, *svg)
	return err
}

// ── Page status ────────────────────────────────────────────────

// ListPageStatuses returns a map of pageID -> status for all pages in a book
// that have an explicit status record. Pages with no record are absent from
// the map and should be treated as 'unread'.
func (s *Store) ListPageStatuses(bookID string) (map[int]string, error) {
	rows, err := s.db.Query(`
		SELECT ps.page_id, ps.status
		FROM page_status ps
		JOIN pages p ON p.id = ps.page_id
		WHERE p.book_id = ?
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[int]string)
	for rows.Next() {
		var pageID int
		var status string
		if err := rows.Scan(&pageID, &status); err != nil {
			return nil, err
		}
		m[pageID] = status
	}
	return m, rows.Err()
}

// UpsertPageStatus sets the read status for a page by its stable ID.
func (s *Store) UpsertPageStatus(pageID int, status string) error {
	_, err := s.db.Exec(`
		INSERT INTO page_status (page_id, status, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(page_id) DO UPDATE SET
			status     = excluded.status,
			updated_at = CURRENT_TIMESTAMP
	`, pageID, status)
	return err
}

// ── Sections ───────────────────────────────────────────────────

// GetTOC returns all section entries for a book ordered by page number.
// Each section references the page it starts on via the stable start_page_id FK.
func (s *Store) GetTOC(bookID string) ([]TocEntry, error) {
	rows, err := s.db.Query(`
		SELECT p.number, s.title
		FROM sections s
		JOIN pages p ON p.id = s.start_page_id
		WHERE s.book_id = ?
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

// syncSection keeps the sections table in sync whenever a page's attribute is
// saved. Must be called inside an open transaction.
func syncSection(tx *sql.Tx, pageID int, bookID, attribute, title string) error {
	if attribute == AttrSection {
		_, err := tx.Exec(`
			INSERT INTO sections (book_id, start_page_id, title)
			VALUES (?, ?, ?)
			ON CONFLICT(book_id, start_page_id) DO UPDATE SET title = excluded.title
		`, bookID, pageID, title)
		return err
	}
	_, err := tx.Exec(`DELETE FROM sections WHERE start_page_id = ?`, pageID)
	return err
}

// ── Book notes ─────────────────────────────────────────────────

// GetBookNote returns the memo body for a book, or an empty string if none exists.
func (s *Store) GetBookNote(bookID string) (string, error) {
	var body string
	err := s.db.QueryRow(`SELECT body FROM notes WHERE book_id = ?`, bookID).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return body, err
}

// UpsertBookNote inserts or updates the memo for a book.
func (s *Store) UpsertBookNote(bookID, body string) error {
	_, err := s.db.Exec(`
		INSERT INTO notes (book_id, body, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(book_id) DO UPDATE SET
			body       = excluded.body,
			updated_at = CURRENT_TIMESTAMP
	`, bookID, body)
	return err
}

// ── Collections ────────────────────────────────────────────────

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
// ON DELETE CASCADE handles collection_books, collection_tags, and collection notes.
func (s *Store) DeleteCollection(id int) error {
	_, err := s.db.Exec(`DELETE FROM collections WHERE id = ?`, id)
	return err
}

// AddBookToCollection adds a book to a collection.
// Returns whether the book was newly added (false means it was already a member).
func (s *Store) AddBookToCollection(collectionID int, bookID string) (added bool, err error) {
	res, err := s.db.Exec(`
		INSERT OR IGNORE INTO collection_books (collection_id, book_id) VALUES (?, ?)
	`, collectionID, bookID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
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
		SELECT b.id, b.title, b.source, b.status, b.file_mtime, b.missing_since
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
		if err := rows.Scan(&b.ID, &b.Title, &b.Source, &b.Status, &b.FileMtime, &b.MissingSince); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// ListUncategorizedBooks returns non-missing books that do not belong to any
// collection, ordered by title.
func (s *Store) ListUncategorizedBooks() ([]Book, error) {
	rows, err := s.db.Query(`
		SELECT id, title, source, status, file_mtime, missing_since
		FROM books
		WHERE missing_since IS NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM collection_books cb WHERE cb.book_id = books.id
		  )
		ORDER BY title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Source, &b.Status, &b.FileMtime, &b.MissingSince); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// CountUncategorizedBooks returns the number of non-missing books that do not
// belong to any collection.
func (s *Store) CountUncategorizedBooks() (int, error) {
	var n int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM books
		WHERE missing_since IS NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM collection_books cb WHERE cb.book_id = books.id
		  )
	`).Scan(&n)
	return n, err
}
