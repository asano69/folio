package store

import (
	"database/sql"
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

-- pages holds scan-derived data plus the user-assigned real book page number.
-- seq is the 1-based position of the image within the CBZ (filename sort order).
-- page_number is the real book page number as printed (TEXT to support roman
-- numerals such as "i", "ii", "iii" used in front matter); NULL when not set.
-- pages.id is stable across re-scans thanks to the merge algorithm in UpsertPages.
CREATE TABLE IF NOT EXISTS pages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id     TEXT    NOT NULL REFERENCES books(id),
    seq         INTEGER NOT NULL,
    filename    TEXT    NOT NULL,
    hash        TEXT    NOT NULL DEFAULT '',
    page_number TEXT,
    UNIQUE(book_id, seq)
);

-- ── Per-page annotations ───────────────────────────────────────

-- One text note per page. Absence of a row means no note has been written.
CREATE TABLE IF NOT EXISTS page_notes (
    page_id    INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
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

-- ── Sections ───────────────────────────────────────────────────

-- A section is a named range within a book. start_page_id and end_page_id are
-- both references to pages.id. end_page_id is nullable (NULL means the user has
-- not set an explicit end). Sections may overlap or nest; no uniqueness constraint
-- is enforced. ON DELETE CASCADE for start_page_id removes the section if its
-- starting page disappears; ON DELETE SET NULL for end_page_id clears the end
-- boundary rather than removing the whole section.
CREATE TABLE IF NOT EXISTS sections (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id       TEXT    NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    start_page_id INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    end_page_id   INTEGER          REFERENCES pages(id) ON DELETE SET NULL,
    title         TEXT    NOT NULL DEFAULT '',
    description   TEXT    NOT NULL DEFAULT '',
    status        TEXT    NOT NULL DEFAULT 'unread'
                  CHECK(status IN ('unread','reading','read','skip'))
);

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
CREATE INDEX IF NOT EXISTS idx_sections_book                ON sections(book_id);
CREATE INDEX IF NOT EXISTS idx_book_collection_members_book ON book_collection_members(book_id);
CREATE INDEX IF NOT EXISTS idx_page_collection_members_page ON page_collection_members(page_id);
`

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

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
