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

CREATE TABLE IF NOT EXISTS pages (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id  TEXT    NOT NULL REFERENCES books(id),
    number   INTEGER NOT NULL,
    filename TEXT    NOT NULL,
    hash     TEXT    NOT NULL DEFAULT '',
    UNIQUE(book_id, number)
);

-- Book-level thumbnail.
CREATE TABLE IF NOT EXISTS thumbnails (
    book_id    TEXT PRIMARY KEY REFERENCES books(id),
    data       BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Page-level thumbnail, keyed by page_hash for re-scan safety.
CREATE TABLE IF NOT EXISTS page_thumbnails (
    book_id    TEXT NOT NULL REFERENCES books(id),
    page_hash  TEXT NOT NULL,
    data       BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(book_id, page_hash)
);

-- Per-page notes including optional SVG drawing.
-- Keyed by (book_id, page_hash) for re-scan safety.
-- svg_drawing holds raw SVG markup; NULL when no drawing exists.
CREATE TABLE IF NOT EXISTS notes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id     TEXT NOT NULL REFERENCES books(id),
    page_hash   TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    attribute   TEXT NOT NULL DEFAULT '',
    body        TEXT NOT NULL DEFAULT '',
    svg_drawing TEXT,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(book_id, page_hash)
);

-- Per-page read status, keyed by page_hash for re-scan safety.
CREATE TABLE IF NOT EXISTS page_status (
    book_id    TEXT NOT NULL REFERENCES books(id),
    page_hash  TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'unread'
               CHECK(status IN ('unread','reading','read','skip')),
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(book_id, page_hash)
);

-- OCR text per page, keyed by page_hash for re-scan safety.
CREATE TABLE IF NOT EXISTS page_ocr (
    book_id    TEXT NOT NULL REFERENCES books(id),
    page_hash  TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(book_id, page_hash)
);

-- Sections as independent entities derived from notes where attribute = 'section'.
-- end_page is not stored; it is derived as the next section's start_page - 1.
-- status is preserved across rebuilds (ON CONFLICT only updates title).
CREATE TABLE IF NOT EXISTS sections (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id    TEXT    NOT NULL REFERENCES books(id),
    title      TEXT    NOT NULL DEFAULT '',
    start_page INTEGER NOT NULL,
    status     TEXT    NOT NULL DEFAULT 'unread'
               CHECK(status IN ('unread','reading','read','skip')),
    UNIQUE(book_id, start_page)
);

-- Book-level memo.
CREATE TABLE IF NOT EXISTS book_notes (
    book_id    TEXT PRIMARY KEY REFERENCES books(id),
    body       TEXT NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Collection-level memo.
CREATE TABLE IF NOT EXISTS collection_notes (
    collection_id INTEGER PRIMARY KEY REFERENCES collections(id),
    body          TEXT NOT NULL DEFAULT '',
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tags are scoped per entity type and not shared across scopes.
-- color is stored as a CSS hex string e.g. '#ff0000'.
CREATE TABLE IF NOT EXISTS tags (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    scope TEXT    NOT NULL CHECK(scope IN ('book','page','note','collection')),
    name  TEXT    NOT NULL,
    color TEXT    NOT NULL DEFAULT '#888888',
    UNIQUE(scope, name)
);

CREATE TABLE IF NOT EXISTS book_tags (
    book_id TEXT    NOT NULL REFERENCES books(id),
    tag_id  INTEGER NOT NULL REFERENCES tags(id),
    PRIMARY KEY(book_id, tag_id)
);

-- Keyed by page_hash for re-scan safety.
CREATE TABLE IF NOT EXISTS page_tags (
    book_id   TEXT    NOT NULL REFERENCES books(id),
    page_hash TEXT    NOT NULL,
    tag_id    INTEGER NOT NULL REFERENCES tags(id),
    PRIMARY KEY(book_id, page_hash, tag_id)
);

CREATE TABLE IF NOT EXISTS note_tags (
    note_id INTEGER NOT NULL REFERENCES notes(id),
    tag_id  INTEGER NOT NULL REFERENCES tags(id),
    PRIMARY KEY(note_id, tag_id)
);

CREATE TABLE IF NOT EXISTS collection_tags (
    collection_id INTEGER NOT NULL REFERENCES collections(id),
    tag_id        INTEGER NOT NULL REFERENCES tags(id),
    PRIMARY KEY(collection_id, tag_id)
);

CREATE TABLE IF NOT EXISTS collections (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS collection_books (
    collection_id INTEGER NOT NULL REFERENCES collections(id),
    book_id       TEXT    NOT NULL REFERENCES books(id),
    PRIMARY KEY(collection_id, book_id)
);

-- Indexes for "find all entities with tag X" queries.
CREATE INDEX IF NOT EXISTS idx_book_tags_tag       ON book_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_page_tags_tag       ON page_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_note_tags_tag       ON note_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_collection_tags_tag ON collection_tags(tag_id);

-- Index for listing sections by book (PK is id, not book_id).
CREATE INDEX IF NOT EXISTS idx_sections_book ON sections(book_id);
`

// migrations contains ALTER TABLE statements that cannot be expressed as
// CREATE TABLE IF NOT EXISTS. Each is executed once and the error is discarded
// because SQLite has no "ADD COLUMN IF NOT EXISTS"; a duplicate-column error
// simply means the migration already ran on a previous startup.
var migrations = []string{
	`ALTER TABLE books ADD COLUMN file_mtime INTEGER NOT NULL DEFAULT 0`,
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

	applyMigrations(db)

	return &Store{db: db}, nil
}

func applyMigrations(db *sql.DB) {
	for _, m := range migrations {
		db.Exec(m) // error intentionally ignored; see migrations comment above
	}
}

func (s *Store) Close() error {
	return s.db.Close()
}
