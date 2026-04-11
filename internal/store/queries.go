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

// Image is the DB representation of a single scanned image inside a CBZ.
type Image struct {
	ID       int
	BookID   string
	Number   int
	Filename string
	Hash     string
}

// Note holds user-authored metadata for a single image.
// ID is the integer primary key, used as a stable reference for note_tags.
// SvgDrawing holds raw SVG markup; nil when no drawing has been saved.
// PageHash is the SHA-256 of the image's uncompressed bytes, which
// remains stable across re-scans and CBZ image deletions.
type Note struct {
	ID         int
	BookID     string
	PageHash   string
	Title      string
	Attribute  string
	Body       string
	SvgDrawing *string
	UpdatedAt  string
}

// TocEntry is a single entry in the table of contents derived from section-attributed images.
type TocEntry struct {
	PageNum int
	Title   string
}

// GetTOC returns all section-attributed images for a book, ordered by page number.
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

// UpsertBook inserts a new book or updates its title, source, file_mtime, and
// clears missing_since if it already exists. Status is intentionally excluded
// from the UPDATE so that user-set status is preserved across re-scans.
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
// CBZ file was not found during a scan. It is a no-op if missing_since is
// already set, so the original disappearance time is preserved across scans.
func (s *Store) MarkBookMissing(id string) error {
	_, err := s.db.Exec(`
		UPDATE books SET missing_since = CURRENT_TIMESTAMP
		WHERE id = ? AND missing_since IS NULL
	`, id)
	return err
}

// UpsertImages replaces all image records for a book, then rebuilds the sections
// table so that start_page values stay correct if page numbers have shifted.
func (s *Store) UpsertImages(bookID string, entries []storage.ImageEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM pages WHERE book_id = ?`, bookID); err != nil {
		return err
	}

	for _, e := range entries {
		if _, err := tx.Exec(`
			INSERT INTO pages (book_id, number, filename, hash)
			VALUES (?, ?, ?, ?)
		`, bookID, e.Number, e.Filename, e.Hash); err != nil {
			return err
		}
	}

	if err := rebuildSections(tx, bookID); err != nil {
		return err
	}

	return tx.Commit()
}

// rebuildSections re-derives the sections table from notes where attribute = 'section'.
// Sections whose source image no longer exists are removed. Existing section status
// values are preserved via ON CONFLICT DO UPDATE (only title is overwritten).
func rebuildSections(tx *sql.Tx, bookID string) error {
	_, err := tx.Exec(`
		DELETE FROM sections
		WHERE book_id = ?
		  AND start_page NOT IN (
		      SELECT p.number
		      FROM notes n
		      JOIN pages p ON p.book_id = n.book_id AND p.hash = n.page_hash
		      WHERE n.book_id = ? AND n.attribute = 'section'
		  )
	`, bookID, bookID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO sections (book_id, title, start_page)
		SELECT n.book_id, n.title, p.number
		FROM notes n
		JOIN pages p ON p.book_id = n.book_id AND p.hash = n.page_hash
		WHERE n.book_id = ? AND n.attribute = 'section'
		ON CONFLICT(book_id, start_page) DO UPDATE SET title = excluded.title
	`, bookID)
	return err
}

// UpsertThumbnail inserts or replaces a book thumbnail.
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

// HasImageThumbnail reports whether a thumbnail exists for the given image.
func (s *Store) HasImageThumbnail(bookID, pageHash string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM page_thumbnails WHERE book_id = ? AND page_hash = ?`,
		bookID, pageHash,
	).Scan(&count)
	return count > 0, err
}

// UpsertImageThumbnail inserts or replaces an image-level thumbnail.
func (s *Store) UpsertImageThumbnail(bookID, pageHash string, data []byte) error {
	_, err := s.db.Exec(`
		INSERT INTO page_thumbnails (book_id, page_hash, data)
		VALUES (?, ?, ?)
		ON CONFLICT(book_id, page_hash) DO UPDATE SET
			data       = excluded.data,
			created_at = CURRENT_TIMESTAMP
	`, bookID, pageHash, data)
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
		FROM books
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

// ListBooksUnderPath returns books whose source path is under the given
// directory. Used by partial scans to restrict missing-book detection to
// the scanned subtree only.
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

// ListImages returns all images for a book ordered by number.
func (s *Store) ListImages(bookID string) ([]Image, error) {
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

	var images []Image
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.BookID, &img.Number, &img.Filename, &img.Hash); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

// GetCoverImage returns the first image of a book, or nil if none exists.
func (s *Store) GetCoverImage(bookID string) (*Image, error) {
	var img Image
	err := s.db.QueryRow(`
		SELECT id, book_id, number, filename, hash
		FROM pages
		WHERE book_id = ?
		ORDER BY number
		LIMIT 1
	`, bookID).Scan(&img.ID, &img.BookID, &img.Number, &img.Filename, &img.Hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &img, err
}

// GetNote returns the note for an image identified by its hash, or a zero-value
// Note if none exists.
func (s *Store) GetNote(bookID, pageHash string) (Note, error) {
	var n Note
	err := s.db.QueryRow(`
		SELECT id, book_id, page_hash, title, attribute, body, svg_drawing, updated_at
		FROM notes
		WHERE book_id = ? AND page_hash = ?
	`, bookID, pageHash).Scan(
		&n.ID, &n.BookID, &n.PageHash,
		&n.Title, &n.Attribute, &n.Body,
		&n.SvgDrawing, &n.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Note{BookID: bookID, PageHash: pageHash}, nil
	}
	return n, err
}

// UpsertNote inserts or updates the text fields of an image note (title, attribute,
// body), then synchronises the sections table. svg_drawing is intentionally
// excluded so that saving text annotations never clobbers an existing drawing.
// Use UpsertDrawing to update the SVG drawing independently.
func (s *Store) UpsertNote(n Note) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO notes (book_id, page_hash, title, attribute, body, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(book_id, page_hash) DO UPDATE SET
			title      = excluded.title,
			attribute  = excluded.attribute,
			body       = excluded.body,
			updated_at = CURRENT_TIMESTAMP
	`, n.BookID, n.PageHash, n.Title, n.Attribute, n.Body)
	if err != nil {
		return err
	}

	if err := syncSection(tx, n.BookID, n.PageHash, n.Attribute, n.Title); err != nil {
		return err
	}

	return tx.Commit()
}

// UpsertDrawing inserts or updates only the svg_drawing field of an image note.
// Passing nil clears an existing drawing. Text fields (title, attribute, body)
// are not touched.
func (s *Store) UpsertDrawing(bookID, pageHash string, svg *string) error {
	_, err := s.db.Exec(`
		INSERT INTO notes (book_id, page_hash, svg_drawing, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(book_id, page_hash) DO UPDATE SET
			svg_drawing = excluded.svg_drawing,
			updated_at  = CURRENT_TIMESTAMP
	`, bookID, pageHash, svg)
	return err
}

// syncSection keeps the sections table in sync with notes where attribute = 'section'.
// Called within a transaction whenever a note is saved.
// If the image hash cannot be resolved to a page number the sync is skipped silently.
func syncSection(tx *sql.Tx, bookID, pageHash, attribute, title string) error {
	var pageNum int
	err := tx.QueryRow(
		`SELECT number FROM pages WHERE book_id = ? AND hash = ?`, bookID, pageHash,
	).Scan(&pageNum)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if attribute == AttrSection {
		_, err = tx.Exec(`
			INSERT INTO sections (book_id, title, start_page)
			VALUES (?, ?, ?)
			ON CONFLICT(book_id, start_page) DO UPDATE SET title = excluded.title
		`, bookID, title, pageNum)
	} else {
		_, err = tx.Exec(
			`DELETE FROM sections WHERE book_id = ? AND start_page = ?`, bookID, pageNum,
		)
	}
	return err
}

// HasImages reports whether any images are registered for the given book.
func (s *Store) HasImages(bookID string) (bool, error) {
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

// AddBookToCollection adds a book to a collection.
// Reports whether the book was newly added (false means it was already a member).
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

// CountAllBooks returns the number of non-missing books in the library.
func (s *Store) CountAllBooks() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM books WHERE missing_since IS NULL`).Scan(&n)
	return n, err
}

// ListNotesByBook returns all notes for a book keyed by image hash.
// Used to avoid N+1 queries when rendering an image grid.
func (s *Store) ListNotesByBook(bookID string) (map[string]Note, error) {
	rows, err := s.db.Query(`
		SELECT id, book_id, page_hash, title, attribute, body, svg_drawing, updated_at
		FROM notes WHERE book_id = ?
	`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make(map[string]Note)
	for rows.Next() {
		var n Note
		if err := rows.Scan(
			&n.ID, &n.BookID, &n.PageHash,
			&n.Title, &n.Attribute, &n.Body,
			&n.SvgDrawing, &n.UpdatedAt,
		); err != nil {
			return nil, err
		}
		notes[n.PageHash] = n
	}
	return notes, rows.Err()
}

// ListImageHashesWithThumbnails returns the set of image hashes that have a
// stored thumbnail for the given book, allowing a single query instead of N.
func (s *Store) ListImageHashesWithThumbnails(bookID string) (map[string]bool, error) {
	rows, err := s.db.Query(
		`SELECT page_hash FROM page_thumbnails WHERE book_id = ?`, bookID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	set := make(map[string]bool)
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		set[h] = true
	}
	return set, rows.Err()
}

// GetImageThumbnail returns the JPEG thumbnail bytes for an image, or nil if not found.
func (s *Store) GetImageThumbnail(bookID, pageHash string) ([]byte, error) {
	var data []byte
	err := s.db.QueryRow(
		`SELECT data FROM page_thumbnails WHERE book_id = ? AND page_hash = ?`,
		bookID, pageHash,
	).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return data, err
}

// CountPages returns the total number of pages registered for a book.
func (s *Store) CountPages(bookID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pages WHERE book_id = ?`, bookID).Scan(&n)
	return n, err
}

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

// ListPageStatuses returns a map of page_hash -> status for all pages in a book
// that have an explicit status record. Pages with no record are absent from the map.
func (s *Store) ListPageStatuses(bookID string) (map[string]string, error) {
	rows, err := s.db.Query(
		`SELECT page_hash, status FROM page_status WHERE book_id = ?`, bookID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var hash, status string
		if err := rows.Scan(&hash, &status); err != nil {
			return nil, err
		}
		m[hash] = status
	}
	return m, rows.Err()
}

// UpsertPageStatus sets the read status for a page identified by its content hash.
func (s *Store) UpsertPageStatus(bookID, pageHash, status string) error {
	_, err := s.db.Exec(`
		INSERT INTO page_status (book_id, page_hash, status, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(book_id, page_hash) DO UPDATE SET
			status     = excluded.status,
			updated_at = CURRENT_TIMESTAMP
	`, bookID, pageHash, status)
	return err
}

// ListBookIDsWithThumbnails returns the set of book IDs that have a stored
// thumbnail. Use this instead of calling HasThumbnail per book to avoid N+1
// queries when rendering a book grid.
func (s *Store) ListBookIDsWithThumbnails() (map[string]bool, error) {
	rows, err := s.db.Query(`SELECT book_id FROM thumbnails`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	set := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		set[id] = true
	}
	return set, rows.Err()
}

// GetImageByHash returns the image matching the given hash, or nil if not found.
func (s *Store) GetImageByHash(bookID, pageHash string) (*Image, error) {
	var img Image
	err := s.db.QueryRow(`
		SELECT id, book_id, number, filename, hash
		FROM pages
		WHERE book_id = ? AND hash = ?
	`, bookID, pageHash).Scan(&img.ID, &img.BookID, &img.Number, &img.Filename, &img.Hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &img, err
}
