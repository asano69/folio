package store

import (
	"database/sql"
	"errors"

	"folio/internal/storage"
)

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
//
// ID is stable across re-scans: UpsertPages uses a merge algorithm
// (hash-first, then position) to preserve IDs even when the CBZ changes.
//
// Seq is the 1-based position of the image within the CBZ (filename sort
// order). It is NOT the real book page number.
//
// PageNumber is the real book page number as printed (e.g. "42", "iv").
// It is TEXT to support roman numerals. NULL when not assigned by the user.
type Page struct {
	ID         int
	BookID     string
	Seq        int
	Filename   string
	Hash       string
	PageNumber *string
}

// Section is the DB representation of a named page range within a book.
// Sections may overlap and nest freely; no uniqueness is enforced.
// EndPageID is nil when the user has not set an explicit end boundary.
type Section struct {
	ID          int
	BookID      string
	StartPageID int
	EndPageID   *int
	Title       string
	Description string
	Status      string
}

// TocEntry is a single entry in the table of contents derived from sections.
// StartSeq is the seq of the section-start page. EndSeq is the seq of the
// end page, or nil when end_page_id is NULL.
type TocEntry struct {
	SectionID   int
	StartSeq    int
	EndSeq      *int
	Title       string
	Description string
	Status      string
}

// PageNote holds the user-editable text annotation for a single page.
type PageNote struct {
	Body string
}

// BookCollection is the DB representation of a named group of books.
// BookCount is populated by ListBookCollections.
type BookCollection struct {
	ID          int
	Name        string
	Color       string
	Description string
	BookCount   int
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
//  2. Match remaining entries by seq: a page whose content was replaced
//     in-place keeps its ID; only its hash (and filename) is updated.
//
// Pages with no match are inserted as new rows.
// Existing pages with no match are deleted; ON DELETE CASCADE propagates to
// page_notes, page_status, page_drawings, page_ocr, and sections.
//
// page_number is never touched by this function; user-assigned values survive
// re-scans as long as the page ID is preserved by the merge.
func (s *Store) UpsertPages(bookID string, entries []storage.ImageEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Load existing pages for this book.
	rows, err := tx.Query(
		`SELECT id, seq, hash FROM pages WHERE book_id = ? ORDER BY seq`, bookID,
	)
	if err != nil {
		return err
	}
	type existingPage struct {
		id, seq int
		hash    string
	}
	var existing []existingPage
	for rows.Next() {
		var p existingPage
		if err := rows.Scan(&p.id, &p.seq, &p.hash); err != nil {
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

	// Pass 2: match remaining entries by seq (content replaced in-place).
	for newIdx, entry := range entries {
		if usedNew[newIdx] {
			continue
		}
		for exIdx, ex := range existing {
			if !usedExisting[exIdx] && ex.seq == entry.Seq {
				matches[ex.id] = newIdx
				usedExisting[exIdx] = true
				usedNew[newIdx] = true
				break
			}
		}
	}

	// Update matched pages: seq, filename, or hash may have changed.
	// page_number is intentionally excluded; it is user-managed.
	for exID, newIdx := range matches {
		entry := entries[newIdx]
		if _, err := tx.Exec(
			`UPDATE pages SET seq = ?, filename = ?, hash = ? WHERE id = ?`,
			entry.Seq, entry.Filename, entry.Hash, exID,
		); err != nil {
			return err
		}
	}

	// Insert truly new pages (no existing page matched by hash or seq).
	for newIdx, entry := range entries {
		if usedNew[newIdx] {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO pages (book_id, seq, filename, hash) VALUES (?, ?, ?, ?)`,
			bookID, entry.Seq, entry.Filename, entry.Hash,
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

// ListPages returns all pages for a book ordered by seq.
func (s *Store) ListPages(bookID string) ([]Page, error) {
	rows, err := s.db.Query(`
		SELECT id, book_id, seq, filename, hash, page_number
		FROM pages WHERE book_id = ? ORDER BY seq
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.BookID, &p.Seq, &p.Filename, &p.Hash, &p.PageNumber); err != nil {
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
		SELECT id, book_id, seq, filename, hash, page_number
		FROM pages WHERE id = ?
	`, pageID).Scan(&p.ID, &p.BookID, &p.Seq, &p.Filename, &p.Hash, &p.PageNumber)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// GetCoverPage returns the first page (lowest seq) of a book, or nil if none exists.
func (s *Store) GetCoverPage(bookID string) (*Page, error) {
	var p Page
	err := s.db.QueryRow(`
		SELECT id, book_id, seq, filename, hash, page_number
		FROM pages WHERE book_id = ? ORDER BY seq LIMIT 1
	`, bookID).Scan(&p.ID, &p.BookID, &p.Seq, &p.Filename, &p.Hash, &p.PageNumber)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// GetPageByNumber returns the page with the lowest seq whose page_number matches
// within the given book. Returns nil if no page carries that number.
func (s *Store) GetPageByNumber(bookID, pageNumber string) (*Page, error) {
	var p Page
	err := s.db.QueryRow(`
		SELECT id, book_id, seq, filename, hash, page_number
		FROM pages WHERE book_id = ? AND page_number = ?
		ORDER BY seq LIMIT 1
	`, bookID, pageNumber).Scan(&p.ID, &p.BookID, &p.Seq, &p.Filename, &p.Hash, &p.PageNumber)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

// UpdatePageNumber sets or clears the real book page number for a page.
// Pass nil to clear an existing value.
func (s *Store) UpdatePageNumber(pageID int, pageNumber *string) error {
	_, err := s.db.Exec(`UPDATE pages SET page_number = ? WHERE id = ?`, pageNumber, pageID)
	return err
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

// ── Page notes ─────────────────────────────────────────────────

// GetPageNote returns the note for a page, or a zero-value PageNote if none exists.
func (s *Store) GetPageNote(pageID int) (PageNote, error) {
	var note PageNote
	err := s.db.QueryRow(
		`SELECT body FROM page_notes WHERE page_id = ?`, pageID,
	).Scan(&note.Body)
	if errors.Is(err, sql.ErrNoRows) {
		return PageNote{}, nil
	}
	return note, err
}

// UpsertPageNote inserts or updates the note body for a page.
func (s *Store) UpsertPageNote(pageID int, body string) error {
	_, err := s.db.Exec(`
		INSERT INTO page_notes (page_id, body, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(page_id) DO UPDATE SET
			body       = excluded.body,
			updated_at = CURRENT_TIMESTAMP
	`, pageID, body)
	return err
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

// ListPageStatuses returns a map of pageID → status for all pages in a book
// that have an explicit status record. Pages absent from the map are 'unread'.
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

// ListSections returns all sections for a book in insertion order.
func (s *Store) ListSections(bookID string) ([]Section, error) {
	rows, err := s.db.Query(`
		SELECT id, book_id, start_page_id, end_page_id, title, description, status
		FROM sections WHERE book_id = ? ORDER BY rowid
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sections []Section
	for rows.Next() {
		var sec Section
		if err := rows.Scan(
			&sec.ID, &sec.BookID, &sec.StartPageID, &sec.EndPageID,
			&sec.Title, &sec.Description, &sec.Status,
		); err != nil {
			return nil, err
		}
		sections = append(sections, sec)
	}
	return sections, rows.Err()
}

// GetTOC returns all sections for a book joined with their start/end page seq
// values, ordered by start page seq. This is the primary data source for the
// table of contents in the viewer and bibliography pages.
func (s *Store) GetTOC(bookID string) ([]TocEntry, error) {
	rows, err := s.db.Query(`
		SELECT s.id, p_start.seq, p_end.seq, s.title, s.description, s.status
		FROM sections s
		JOIN pages p_start ON p_start.id = s.start_page_id
		LEFT JOIN pages p_end ON p_end.id = s.end_page_id
		WHERE s.book_id = ?
		ORDER BY p_start.seq
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []TocEntry
	for rows.Next() {
		var e TocEntry
		if err := rows.Scan(
			&e.SectionID, &e.StartSeq, &e.EndSeq,
			&e.Title, &e.Description, &e.Status,
		); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListSectionStartPageIDs returns the set of page IDs that are the start of
// at least one section for the given book. Used to mark pages in overview grids.
func (s *Store) ListSectionStartPageIDs(bookID string) (map[int]bool, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT start_page_id FROM sections WHERE book_id = ?
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[int]bool)
	for rows.Next() {
		var pageID int
		if err := rows.Scan(&pageID); err != nil {
			return nil, err
		}
		m[pageID] = true
	}
	return m, rows.Err()
}

// CreateSection inserts a new section and returns its ID.
func (s *Store) CreateSection(bookID string, startPageID int, endPageID *int, title, description string) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO sections (book_id, start_page_id, end_page_id, title, description)
		VALUES (?, ?, ?, ?, ?)
	`, bookID, startPageID, endPageID, title, description)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateSection replaces all mutable fields of an existing section.
func (s *Store) UpdateSection(id int, startPageID int, endPageID *int, title, description, status string) error {
	_, err := s.db.Exec(`
		UPDATE sections
		SET start_page_id = ?, end_page_id = ?, title = ?, description = ?, status = ?
		WHERE id = ?
	`, startPageID, endPageID, title, description, status, id)
	return err
}

// DeleteSection removes a section by ID.
func (s *Store) DeleteSection(id int) error {
	_, err := s.db.Exec(`DELETE FROM sections WHERE id = ?`, id)
	return err
}

// ── Book notes ─────────────────────────────────────────────────

// GetBookNote returns the memo body for a book, or an empty string if none exists.
func (s *Store) GetBookNote(bookID string) (string, error) {
	var body string
	err := s.db.QueryRow(`SELECT body FROM book_notes WHERE book_id = ?`, bookID).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return body, err
}

// UpsertBookNote inserts or updates the memo for a book.
func (s *Store) UpsertBookNote(bookID, body string) error {
	_, err := s.db.Exec(`
		INSERT INTO book_notes (book_id, body, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(book_id) DO UPDATE SET
			body       = excluded.body,
			updated_at = CURRENT_TIMESTAMP
	`, bookID, body)
	return err
}

// ── Book collections ───────────────────────────────────────────

// ListBookCollections returns all book collections ordered by name, with
// member counts.
func (s *Store) ListBookCollections() ([]BookCollection, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.color, c.description, COUNT(m.book_id)
		FROM book_collections c
		LEFT JOIN book_collection_members m ON m.collection_id = c.id
		GROUP BY c.id
		ORDER BY c.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []BookCollection
	for rows.Next() {
		var c BookCollection
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.Description, &c.BookCount); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// CreateBookCollection inserts a new book collection and returns its ID.
func (s *Store) CreateBookCollection(name string) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO book_collections (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// RenameBookCollection updates the name of an existing book collection.
func (s *Store) RenameBookCollection(id int, name string) error {
	_, err := s.db.Exec(`UPDATE book_collections SET name = ? WHERE id = ?`, name, id)
	return err
}

// DeleteBookCollection removes a book collection and all its memberships.
// ON DELETE CASCADE handles book_collection_members.
func (s *Store) DeleteBookCollection(id int) error {
	_, err := s.db.Exec(`DELETE FROM book_collections WHERE id = ?`, id)
	return err
}

// AddBookToBookCollection adds a book to a book collection.
// Returns whether the book was newly added (false means it was already a member).
func (s *Store) AddBookToBookCollection(collectionID int, bookID string) (added bool, err error) {
	res, err := s.db.Exec(`
		INSERT OR IGNORE INTO book_collection_members (collection_id, book_id) VALUES (?, ?)
	`, collectionID, bookID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// RemoveBookFromBookCollection removes a book from a book collection.
func (s *Store) RemoveBookFromBookCollection(collectionID int, bookID string) error {
	_, err := s.db.Exec(`
		DELETE FROM book_collection_members WHERE collection_id = ? AND book_id = ?
	`, collectionID, bookID)
	return err
}

// ListBooksInBookCollection returns books belonging to a book collection,
// ordered by title.
func (s *Store) ListBooksInBookCollection(collectionID int) ([]Book, error) {
	rows, err := s.db.Query(`
		SELECT b.id, b.title, b.source, b.status, b.file_mtime, b.missing_since
		FROM books b
		JOIN book_collection_members m ON m.book_id = b.id
		WHERE m.collection_id = ?
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
// book collection, ordered by title.
func (s *Store) ListUncategorizedBooks() ([]Book, error) {
	rows, err := s.db.Query(`
		SELECT id, title, source, status, file_mtime, missing_since
		FROM books
		WHERE missing_since IS NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM book_collection_members m WHERE m.book_id = books.id
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
// belong to any book collection.
func (s *Store) CountUncategorizedBooks() (int, error) {
	var n int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM books
		WHERE missing_since IS NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM book_collection_members m WHERE m.book_id = books.id
		  )
	`).Scan(&n)
	return n, err
}
