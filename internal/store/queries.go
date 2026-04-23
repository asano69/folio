package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"folio/internal/storage"

	"github.com/google/uuid"
)

// CentralLibraryID is the fixed UUID of the default library. It cannot be
// deleted or renamed, and implicitly contains all book collections.
const CentralLibraryID = "00000000-0000-7000-8000-000000000000"

// Library is the DB representation of a named group of book collections.
type Library struct {
	ID              string
	Name            string
	CollectionCount int
}

// Book is the DB representation of a book, including extended metadata fields
// that mirror folio.json. Array fields (Author, Translator, Keywords, Links)
// are stored as JSON text in SQLite and decoded on read.
type Book struct {
	ID           string
	Title        string
	Source       string
	Status       string
	FileMtime    int64
	MissingSince *string
	// Extended metadata mirrored from folio.json
	Type         string
	Abstract     string
	Language     string
	Author       []storage.PersonName
	Translator   []storage.PersonName
	OrigTitle    string
	Edition      string
	Volume       string
	Series       string
	SeriesNumber string
	Publisher    string
	Year         string
	Note         string
	Keywords     []string
	ISBN         string
	Links        []string
}

// Page is the DB representation of a single scanned image inside a CBZ.
//
// ID is stable across re-scans: UpsertPages uses a merge algorithm
// (hash-first, then position) to preserve IDs even when the CBZ changes.
type Page struct {
	ID         int
	BookID     string
	Seq        int
	Filename   string
	Hash       string
	PageNumber *string
}

// Section is the DB representation of a named page range within a book.
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
type BookCollection struct {
	ID          string
	Name        string
	Color       string
	Description string
	BookCount   int
	LibraryIDs  string // comma-separated IDs from library_collection_members; empty = Central only
}

// ── JSON helpers for array columns ────────────────────────────

func marshalPersonNames(names []storage.PersonName) string {
	if len(names) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(names)
	return string(b)
}

func unmarshalPersonNames(s string) []storage.PersonName {
	if s == "" || s == "[]" {
		return nil
	}
	var names []storage.PersonName
	_ = json.Unmarshal([]byte(s), &names)
	return names
}

func marshalStrings(strs []string) string {
	if len(strs) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(strs)
	return string(b)
}

func unmarshalStrings(s string) []string {
	if s == "" || s == "[]" {
		return nil
	}
	var strs []string
	_ = json.Unmarshal([]byte(s), &strs)
	return strs
}

// bookSelectSQL is the SELECT column list used by all book-returning queries.
const bookSelectSQL = `
    id, title, source, status, file_mtime, missing_since,
    type, abstract, language, author, translator, origtitle,
    edition, volume, series, series_number, publisher, year,
    note, keywords, isbn, links
`

// ── Books ──────────────────────────────────────────────────────

func (s *Store) UpsertBook(b storage.Book) error {
	_, err := s.db.Exec(`
		INSERT INTO books (
			id, title, source, file_mtime,
			type, abstract, language, author, translator, origtitle,
			edition, volume, series, series_number, publisher, year,
			note, keywords, isbn, links
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title         = excluded.title,
			source        = excluded.source,
			file_mtime    = excluded.file_mtime,
			type          = excluded.type,
			abstract      = excluded.abstract,
			language      = excluded.language,
			author        = excluded.author,
			translator    = excluded.translator,
			origtitle     = excluded.origtitle,
			edition       = excluded.edition,
			volume        = excluded.volume,
			series        = excluded.series,
			series_number = excluded.series_number,
			publisher     = excluded.publisher,
			year          = excluded.year,
			note          = excluded.note,
			keywords      = excluded.keywords,
			isbn          = excluded.isbn,
			links         = excluded.links,
			missing_since = NULL
	`,
		b.ID, b.Title, b.Source, b.FileMtime,
		b.Type, b.Abstract, b.Language,
		marshalPersonNames(b.Author), marshalPersonNames(b.Translator),
		b.OrigTitle, b.Edition, b.Volume, b.Series, b.SeriesNumber,
		b.Publisher, b.Year, b.Note,
		marshalStrings(b.Keywords), b.ISBN, marshalStrings(b.Links),
	)
	return err
}

func (s *Store) UpdateBookMeta(id string, b storage.Book) error {
	_, err := s.db.Exec(`
		UPDATE books SET
			title         = ?,
			type          = ?,
			abstract      = ?,
			language      = ?,
			author        = ?,
			translator    = ?,
			origtitle     = ?,
			edition       = ?,
			volume        = ?,
			series        = ?,
			series_number = ?,
			publisher     = ?,
			year          = ?,
			note          = ?,
			keywords      = ?,
			isbn          = ?,
			links         = ?
		WHERE id = ?
	`,
		b.Title, b.Type, b.Abstract, b.Language,
		marshalPersonNames(b.Author), marshalPersonNames(b.Translator),
		b.OrigTitle, b.Edition, b.Volume, b.Series, b.SeriesNumber,
		b.Publisher, b.Year, b.Note,
		marshalStrings(b.Keywords), b.ISBN, marshalStrings(b.Links),
		id,
	)
	return err
}

// MarkBookMissing sets missing_since for a book whose CBZ was not found.
// It is a no-op when missing_since is already set.
func (s *Store) MarkBookMissing(id string) error {
	_, err := s.db.Exec(`
		UPDATE books SET missing_since = CURRENT_TIMESTAMP
		WHERE id = ? AND missing_since IS NULL
	`, id)
	return err
}

func (s *Store) UpdateBookTitle(id, title string) error {
	_, err := s.db.Exec(`UPDATE books SET title = ? WHERE id = ?`, title, id)
	return err
}

func (s *Store) ListBooks() ([]Book, error) {
	rows, err := s.db.Query(`SELECT` + bookSelectSQL + `FROM books ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookRows(rows)
}

func (s *Store) ListBooksUnderPath(dirPath string) ([]Book, error) {
	rows, err := s.db.Query(
		`SELECT`+bookSelectSQL+`FROM books WHERE source LIKE ? || '/%'`,
		dirPath,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookRows(rows)
}

// ListAllBooksInLibrary returns books associated with the given library.
// Central Library returns all books; other libraries return books from their collections.
func (s *Store) ListAllBooksInLibrary(libraryID string) ([]Book, error) {
	if libraryID == CentralLibraryID {
		return s.ListBooks()
	}
	rows, err := s.db.Query(
		`SELECT`+bookSelectSQL+`FROM books
		 WHERE id IN (
		     SELECT DISTINCT bcm.book_id
		     FROM book_collection_members bcm
		     JOIN library_collection_members lcm ON lcm.collection_id = bcm.collection_id
		     WHERE lcm.library_id = ?
		 )
		 ORDER BY title`,
		libraryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookRows(rows)
}

func (s *Store) GetBook(id string) (*Book, error) {
	rows, err := s.db.Query(
		`SELECT`+bookSelectSQL+`FROM books WHERE id = ?`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	books, err := scanBookRows(rows)
	if err != nil {
		return nil, err
	}
	if len(books) == 0 {
		return nil, nil
	}
	return &books[0], nil
}

func (s *Store) CountAllBooks() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM books WHERE missing_since IS NULL`).Scan(&n)
	return n, err
}

func scanBookRows(rows *sql.Rows) ([]Book, error) {
	var books []Book
	for rows.Next() {
		var (
			id, title, source, status string
			fileMtime                 int64
			missingSince              *string
			typ, abstract, language   string
			authorJSON, transJSON     string
			origTitle, edition        string
			volume, series, seriesNum string
			publisher, year, note     string
			keywordsJSON, isbn        string
			linksJSON                 string
		)
		if err := rows.Scan(
			&id, &title, &source, &status, &fileMtime, &missingSince,
			&typ, &abstract, &language, &authorJSON, &transJSON,
			&origTitle, &edition, &volume, &series, &seriesNum,
			&publisher, &year, &note, &keywordsJSON, &isbn, &linksJSON,
		); err != nil {
			return nil, err
		}
		books = append(books, Book{
			ID:           id,
			Title:        title,
			Source:       source,
			Status:       status,
			FileMtime:    fileMtime,
			MissingSince: missingSince,
			Type:         typ,
			Abstract:     abstract,
			Language:     language,
			Author:       unmarshalPersonNames(authorJSON),
			Translator:   unmarshalPersonNames(transJSON),
			OrigTitle:    origTitle,
			Edition:      edition,
			Volume:       volume,
			Series:       series,
			SeriesNumber: seriesNum,
			Publisher:    publisher,
			Year:         year,
			Note:         note,
			Keywords:     unmarshalStrings(keywordsJSON),
			ISBN:         isbn,
			Links:        unmarshalStrings(linksJSON),
		})
	}
	return books, rows.Err()
}

// ── Pages ──────────────────────────────────────────────────────

// UpsertPages merges the scanned image list into the pages table while
// preserving stable page IDs via hash-first, then position matching.
func (s *Store) UpsertPages(bookID string, entries []storage.ImageEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
	matches := make(map[int]int, len(existing)) // existingPage.id -> entries index

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

	for exID, newIdx := range matches {
		entry := entries[newIdx]
		if _, err := tx.Exec(
			`UPDATE pages SET seq = ?, filename = ?, hash = ? WHERE id = ?`,
			entry.Seq, entry.Filename, entry.Hash, exID,
		); err != nil {
			return err
		}
	}

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

func (s *Store) UpdatePageNumber(pageID int, pageNumber *string) error {
	_, err := s.db.Exec(`UPDATE pages SET page_number = ? WHERE id = ?`, pageNumber, pageID)
	return err
}

func (s *Store) HasPages(bookID string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pages WHERE book_id = ?`, bookID).Scan(&count)
	return count > 0, err
}

func (s *Store) CountPages(bookID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pages WHERE book_id = ?`, bookID).Scan(&n)
	return n, err
}

// ── Page notes ─────────────────────────────────────────────────

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

func (s *Store) GetPageDrawing(pageID int) (string, error) {
	var svg string
	err := s.db.QueryRow(`SELECT svg FROM page_drawings WHERE page_id = ?`, pageID).Scan(&svg)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return svg, err
}

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

func (s *Store) UpdateSection(id int, startPageID int, endPageID *int, title, description, status string) error {
	_, err := s.db.Exec(`
		UPDATE sections
		SET start_page_id = ?, end_page_id = ?, title = ?, description = ?, status = ?
		WHERE id = ?
	`, startPageID, endPageID, title, description, status, id)
	return err
}

func (s *Store) DeleteSection(id int) error {
	_, err := s.db.Exec(`DELETE FROM sections WHERE id = ?`, id)
	return err
}

// ── Libraries ─────────────────────────────────────────────────

func (s *Store) ListLibraries() ([]Library, error) {
	rows, err := s.db.Query(`
		SELECT l.id, l.name,
		  CASE WHEN l.id = ?
		       THEN (SELECT COUNT(*) FROM book_collections)
		       ELSE (SELECT COUNT(*) FROM library_collection_members m WHERE m.library_id = l.id)
		  END AS collection_count
		FROM libraries l
		ORDER BY l.name
	`, CentralLibraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libs []Library
	for rows.Next() {
		var lib Library
		if err := rows.Scan(&lib.ID, &lib.Name, &lib.CollectionCount); err != nil {
			return nil, err
		}
		libs = append(libs, lib)
	}
	return libs, rows.Err()
}

func (s *Store) GetLibrary(id string) (*Library, error) {
	var lib Library
	err := s.db.QueryRow(`
		SELECT l.id, l.name,
		  CASE WHEN l.id = ?
		       THEN (SELECT COUNT(*) FROM book_collections)
		       ELSE (SELECT COUNT(*) FROM library_collection_members m WHERE m.library_id = l.id)
		  END AS collection_count
		FROM libraries l
		WHERE l.id = ?
	`, CentralLibraryID, id).Scan(&lib.ID, &lib.Name, &lib.CollectionCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &lib, err
}

// CreateLibrary inserts a new library with a generated UUID v7 and returns the ID.
func (s *Store) CreateLibrary(name string) (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	idStr := id.String()
	if _, err := s.db.Exec(`INSERT INTO libraries (id, name) VALUES (?, ?)`, idStr, name); err != nil {
		return "", err
	}
	return idStr, nil
}

func (s *Store) RenameLibrary(id string, name string) error {
	if id == CentralLibraryID {
		return errors.New("cannot rename Central Library")
	}
	_, err := s.db.Exec(`UPDATE libraries SET name = ? WHERE id = ?`, name, id)
	return err
}

func (s *Store) DeleteLibrary(id string) error {
	if id == CentralLibraryID {
		return errors.New("cannot delete Central Library")
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM library_collection_members WHERE library_id = ?`, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return errors.New("library still has collections; move or delete them first")
	}
	_, err := s.db.Exec(`DELETE FROM libraries WHERE id = ?`, id)
	return err
}

// ── Book collections ───────────────────────────────────────────

const collectionSelectSQL = `c.id, c.name, c.color, c.description, COUNT(DISTINCT m.book_id), COALESCE(GROUP_CONCAT(DISTINCT lcm.library_id), '')`

func (s *Store) ListBookCollections() ([]BookCollection, error) {
	rows, err := s.db.Query(`
		SELECT ` + collectionSelectSQL + `
		FROM book_collections c
		LEFT JOIN book_collection_members m ON m.collection_id = c.id
		LEFT JOIN library_collection_members lcm ON lcm.collection_id = c.id
		GROUP BY c.id
		ORDER BY c.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookCollectionRows(rows)
}

// ListBookCollectionsInLibrary returns collections belonging to the given library.
// Central Library returns all collections.

func (s *Store) ListBookCollectionsInLibrary(libraryID string) ([]BookCollection, error) {
	if libraryID == CentralLibraryID {
		return s.ListBookCollections()
	}
	rows, err := s.db.Query(`
		SELECT `+collectionSelectSQL+`
		FROM book_collections c
		LEFT JOIN book_collection_members m ON m.collection_id = c.id
		LEFT JOIN library_collection_members lcm ON lcm.collection_id = c.id
		WHERE c.id IN (
		    SELECT collection_id FROM library_collection_members WHERE library_id = ?
		)
		GROUP BY c.id
		ORDER BY c.name
	`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookCollectionRows(rows)
}

func (s *Store) GetBookCollection(id string) (*BookCollection, error) {
	rows, err := s.db.Query(`
		SELECT `+collectionSelectSQL+`
		FROM book_collections c
		LEFT JOIN book_collection_members m ON m.collection_id = c.id
		LEFT JOIN library_collection_members lcm ON lcm.collection_id = c.id
		WHERE c.id = ?
		GROUP BY c.id
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := scanBookCollectionRows(rows)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, nil
	}
	return &cols[0], nil
}

func scanBookCollectionRows(rows *sql.Rows) ([]BookCollection, error) {
	var cols []BookCollection
	for rows.Next() {
		var c BookCollection
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.Description, &c.BookCount, &c.LibraryIDs); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// CreateBookCollection inserts a new collection with a generated UUID v7.
// When libraryID is not Central Library, it is added to that library's membership.
func (s *Store) CreateBookCollection(name string, libraryID string) (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	idStr := id.String()
	if _, err := s.db.Exec(`INSERT INTO book_collections (id, name) VALUES (?, ?)`, idStr, name); err != nil {
		return "", err
	}
	if libraryID != CentralLibraryID && libraryID != "" {
		if _, err := s.db.Exec(
			`INSERT OR IGNORE INTO library_collection_members (library_id, collection_id) VALUES (?, ?)`,
			libraryID, idStr,
		); err != nil {
			return "", err
		}
	}
	return idStr, nil
}

func (s *Store) RenameBookCollection(id string, name string) error {
	_, err := s.db.Exec(`UPDATE book_collections SET name = ? WHERE id = ?`, name, id)
	return err
}

func (s *Store) DeleteBookCollection(id string) error {
	_, err := s.db.Exec(`DELETE FROM book_collections WHERE id = ?`, id)
	return err
}

// AddCollectionToLibrary adds a collection to a library's membership.
// Central Library is a no-op since it implicitly contains all collections.
func (s *Store) AddCollectionToLibrary(libraryID, collectionID string) (bool, error) {
	if libraryID == CentralLibraryID {
		return false, nil
	}
	res, err := s.db.Exec(`
		INSERT OR IGNORE INTO library_collection_members (library_id, collection_id) VALUES (?, ?)
	`, libraryID, collectionID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *Store) RemoveCollectionFromLibrary(libraryID, collectionID string) error {
	if libraryID == CentralLibraryID {
		return nil
	}
	_, err := s.db.Exec(`
		DELETE FROM library_collection_members WHERE library_id = ? AND collection_id = ?
	`, libraryID, collectionID)
	return err
}

func (s *Store) AddBookToBookCollection(collectionID string, bookID string) (added bool, err error) {
	res, err := s.db.Exec(`
		INSERT OR IGNORE INTO book_collection_members (collection_id, book_id) VALUES (?, ?)
	`, collectionID, bookID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *Store) RemoveBookFromBookCollection(collectionID string, bookID string) error {
	_, err := s.db.Exec(`
		DELETE FROM book_collection_members WHERE collection_id = ? AND book_id = ?
	`, collectionID, bookID)
	return err
}

func (s *Store) ListBooksInBookCollection(collectionID string) ([]Book, error) {
	rows, err := s.db.Query(
		`SELECT`+bookSelectSQL+`FROM books b
		 JOIN book_collection_members m ON m.book_id = b.id
		 WHERE m.collection_id = ?
		 ORDER BY b.title`,
		collectionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookRows(rows)
}

func (s *Store) ListUncategorizedBooks() ([]Book, error) {
	rows, err := s.db.Query(
		`SELECT` + bookSelectSQL + `FROM books
		 WHERE missing_since IS NULL
		   AND NOT EXISTS (
		       SELECT 1 FROM book_collection_members m WHERE m.book_id = books.id
		   )
		 ORDER BY title`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBookRows(rows)
}

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
