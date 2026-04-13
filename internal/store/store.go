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

-- pages.id is stable across re-scans. UpsertImages uses a merge algorithm
-- (hash-first, then position) to preserve IDs even when the CBZ changes.
-- title and attribute are page-level metadata edited by the user.
CREATE TABLE IF NOT EXISTS pages (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id    TEXT    NOT NULL REFERENCES books(id),
    number     INTEGER NOT NULL,
    filename   TEXT    NOT NULL,
    hash       TEXT    NOT NULL DEFAULT '',
    title      TEXT    NOT NULL DEFAULT '',
    attribute  TEXT    NOT NULL DEFAULT '',
    UNIQUE(book_id, number)
);

CREATE TABLE IF NOT EXISTS collections (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Sections reference pages by stable pages.id so they survive re-scans
-- without requiring a rebuild step.
CREATE TABLE IF NOT EXISTS sections (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id       TEXT    NOT NULL REFERENCES books(id),
    start_page_id INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    title         TEXT    NOT NULL DEFAULT '',
    status        TEXT    NOT NULL DEFAULT 'unread'
                  CHECK(status IN ('unread','reading','read','skip')),
    UNIQUE(book_id, start_page_id)
);

-- Unified notes table: one text body per page, book, or collection.
-- Exactly one of page_id, book_id, collection_id is non-NULL (enforced by CHECK).
-- The UNIQUE constraint on each nullable FK guarantees at most one note per entity;
-- SQLite treats NULLs as distinct so multiple NULL values are allowed.
CREATE TABLE IF NOT EXISTS notes (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id       INTEGER UNIQUE REFERENCES pages(id)       ON DELETE CASCADE,
    book_id       TEXT    UNIQUE REFERENCES books(id),
    collection_id INTEGER UNIQUE REFERENCES collections(id) ON DELETE CASCADE,
    body          TEXT    NOT NULL DEFAULT '',
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK(
        (page_id IS NOT NULL) +
        (book_id IS NOT NULL) +
        (collection_id IS NOT NULL) = 1
    )
);

-- SVG annotation drawings stored separately from text notes.
-- One drawing per page; NULL drawing is represented by the absence of a row.
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

-- OCR text per page, keyed by stable page ID.
CREATE TABLE IF NOT EXISTS page_ocr (
    page_id    INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    body       TEXT    NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);


-- Tags scoped per entity type; not shared across scopes.
-- color is stored as a CSS hex string e.g. '#ff0000'.
CREATE TABLE IF NOT EXISTS tags (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    scope TEXT    NOT NULL CHECK(scope IN ('book','page','note','collection')),
    name  TEXT    NOT NULL,
    color TEXT    NOT NULL DEFAULT '#888888',
    UNIQUE(scope, name)
);

CREATE TABLE IF NOT EXISTS book_tags (
    book_id TEXT    NOT NULL REFERENCES books(id)  ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id)   ON DELETE CASCADE,
    PRIMARY KEY(book_id, tag_id)
);

CREATE TABLE IF NOT EXISTS page_tags (
    page_id INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY(page_id, tag_id)
);

CREATE TABLE IF NOT EXISTS note_tags (
    note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY(note_id, tag_id)
);

CREATE TABLE IF NOT EXISTS collection_tags (
    collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    tag_id        INTEGER NOT NULL REFERENCES tags(id)        ON DELETE CASCADE,
    PRIMARY KEY(collection_id, tag_id)
);

CREATE TABLE IF NOT EXISTS collection_books (
    collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    book_id       TEXT    NOT NULL REFERENCES books(id),
    PRIMARY KEY(collection_id, book_id)
);

CREATE INDEX IF NOT EXISTS idx_pages_book          ON pages(book_id);
CREATE INDEX IF NOT EXISTS idx_sections_book       ON sections(book_id);
CREATE INDEX IF NOT EXISTS idx_book_tags_tag       ON book_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_page_tags_tag       ON page_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_note_tags_tag       ON note_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_collection_tags_tag ON collection_tags(tag_id);
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
