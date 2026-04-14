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

-- pages holds scan-derived data only. User-editable data lives in separate
-- tables keyed by pages.id, which is stable across re-scans thanks to the
-- merge algorithm in UpsertPages.
--
-- seq is the 1-based position of the image within the CBZ (filename sort
-- order). It is NOT the real book page number; use page_labels for that.
CREATE TABLE IF NOT EXISTS pages (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id    TEXT    NOT NULL REFERENCES books(id),
    seq        INTEGER NOT NULL,
    filename   TEXT    NOT NULL,
    hash       TEXT    NOT NULL DEFAULT '',
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

-- Marks a page as the start of a named section.
-- Absence of a row means the page is not a section start.
CREATE TABLE IF NOT EXISTS page_sections (
    page_id     INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    title       TEXT    NOT NULL DEFAULT '',
    description TEXT    NOT NULL DEFAULT '',
    status      TEXT    NOT NULL DEFAULT 'unread'
                CHECK(status IN ('unread','reading','read','skip')),
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Real book page number labels for a scanned image. A single image can carry
-- multiple labels (e.g. a spread covering pages 32 and 33 has two rows).
-- label is TEXT to support roman numerals (i, ii, iii...) used in front matter.
CREATE TABLE IF NOT EXISTS page_labels (
    page_id    INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    label      TEXT    NOT NULL,
    PRIMARY KEY(page_id, label)
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
CREATE INDEX IF NOT EXISTS idx_page_sections_page           ON page_sections(page_id);
CREATE INDEX IF NOT EXISTS idx_page_labels_label            ON page_labels(label);
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
