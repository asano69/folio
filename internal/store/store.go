package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

const schema = `
-- Enable foreign key enforcement. Must be set per connection.
PRAGMA foreign_keys = ON;

-- WAL mode allows concurrent reads during writes.
PRAGMA journal_mode = WAL;

-- ── Core entities ──────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS books (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    source        TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'unread'
                  CHECK(status IN ('unread','reading','read','skip')),
    file_mtime    INTEGER NOT NULL DEFAULT 0,
    missing_since DATETIME,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- pages holds scan-derived data only. User-editable data (note title,
-- note body, section marking) lives in separate tables keyed by pages.id,
-- which is stable across re-scans thanks to the merge algorithm in UpsertPages.
CREATE TABLE IF NOT EXISTS pages (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id    TEXT    NOT NULL REFERENCES books(id),
    number     INTEGER NOT NULL,
    filename   TEXT    NOT NULL,
    hash       TEXT    NOT NULL DEFAULT '',
    UNIQUE(book_id, number)
);

-- ── Per-page annotations ───────────────────────────────────────

-- One text note per page. Absence of a row means no note has been written.
CREATE TABLE IF NOT EXISTS page_notes (
    page_id    INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    title      TEXT    NOT NULL DEFAULT '',
    body       TEXT    NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- SVG annotation drawing per page. Absence of a row means no drawing exists.
CREATE TABLE IF NOT EXISTS page_drawings (
    page_id    INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    svg        TEXT    NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Per-page read status.
CREATE TABLE IF NOT EXISTS page_status (
    page_id    INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    status     TEXT    NOT NULL DEFAULT 'unread'
               CHECK(status IN ('unread','reading','read','skip')),
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- OCR text per page.
CREATE TABLE IF NOT EXISTS page_ocr (
    page_id    INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    body       TEXT    NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Marks a page as the start of a named section.
-- Absence of a row means the page is not a section start.
-- The section title is independent from the page note title.
-- description holds optional free-form text about the section.
CREATE TABLE IF NOT EXISTS page_sections (
    page_id     INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    title       TEXT    NOT NULL DEFAULT '',
    description TEXT    NOT NULL DEFAULT '',
    status      TEXT    NOT NULL DEFAULT 'unread'
                CHECK(status IN ('unread','reading','read','skip')),
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- section_ranges derives the end page for each section using the start page of
-- the next section within the same book. The final section in a book extends to
-- the last page. end_page is not stored because it is fully determined by the
-- ordering of start pages; storing it would require keeping adjacent rows in
-- sync on every insert, update, or delete.
CREATE VIEW IF NOT EXISTS section_ranges AS
SELECT
    ps.page_id,
    ps.title,
    ps.description,
    ps.status,
    p.book_id,
    p.number AS start_page,
    COALESCE(
        LEAD(p.number) OVER (PARTITION BY p.book_id ORDER BY p.number) - 1,
        (SELECT MAX(p2.number) FROM pages p2 WHERE p2.book_id = p.book_id)
    ) AS end_page
FROM page_sections ps
JOIN pages p ON p.id = ps.page_id;

-- ── Per-book annotations ───────────────────────────────────────

-- One memo per book.
CREATE TABLE IF NOT EXISTS book_notes (
    book_id    TEXT PRIMARY KEY REFERENCES books(id),
    body       TEXT NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ── Collections ────────────────────────────────────────────────

-- Named groups of books (used for sidebar navigation and filtering).
CREATE TABLE IF NOT EXISTS book_collections (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    color       TEXT NOT NULL DEFAULT '#888888',
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Named groups of pages (for cross-book page organization).
CREATE TABLE IF NOT EXISTS page_collections (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    color       TEXT NOT NULL DEFAULT '#888888',
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ── Collection members ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS book_collection_members (
    collection_id INTEGER NOT NULL REFERENCES book_collections(id) ON DELETE CASCADE,
    book_id       TEXT    NOT NULL REFERENCES books(id),
    PRIMARY KEY(collection_id, book_id)
);

CREATE TABLE IF NOT EXISTS page_collection_members (
    collection_id INTEGER NOT NULL REFERENCES page_collections(id) ON DELETE CASCADE,
    page_id       INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    PRIMARY KEY(collection_id, page_id)
);

-- ── Indexes ────────────────────────────────────────────────────

CREATE INDEX IF NOT EXISTS idx_pages_book                   ON pages(book_id);
CREATE INDEX IF NOT EXISTS idx_page_sections_book           ON page_sections(page_id);
CREATE INDEX IF NOT EXISTS idx_book_collection_members_book ON book_collection_members(book_id);
CREATE INDEX IF NOT EXISTS idx_page_collection_members_page ON page_collection_members(page_id);
`

// migrations runs once per Open call. Each entry is applied only when the
// condition query returns no rows (i.e. the change has not yet been applied).
// This handles databases created before the schema was updated.
var migrations = []struct {
	condition string // returns a row if the migration has already been applied
	statement string
}{
	{
		// Add description column to page_sections if it does not yet exist.
		condition: `SELECT 1 FROM pragma_table_info('page_sections') WHERE name = 'description'`,
		statement: `ALTER TABLE page_sections ADD COLUMN description TEXT NOT NULL DEFAULT ''`,
	},
}

func Open(dataPath string) (*Store, error) {
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataPath, "folio.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// SQLite only supports one concurrent writer. Limiting to a single
	// connection avoids "database is locked" errors under concurrent requests.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return &Store{db: db}, nil
}

// applyMigrations runs each pending migration exactly once. A migration is
// considered pending when its condition query returns no rows.
func applyMigrations(db *sql.DB) error {
	for _, m := range migrations {
		var applied int
		err := db.QueryRow(m.condition).Scan(&applied)
		if errors.Is(err, sql.ErrNoRows) {
			if _, execErr := db.Exec(m.statement); execErr != nil {
				return fmt.Errorf("migration %q: %w", m.statement, execErr)
			}
		} else if err != nil {
			return fmt.Errorf("migration condition %q: %w", m.condition, err)
		}
		// err == nil means the condition row exists; migration already applied.
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
