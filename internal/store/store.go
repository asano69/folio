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

// centralLibraryUUID is the fixed UUID for Central Library.
// Keep in sync with CentralLibraryID in queries.go.
const centralLibraryUUID = "00000000-0000-7000-8000-000000000000"

const schema = `
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

-- ── Libraries ──────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS libraries (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ── Core entities ──────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS books (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    source        TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'unread'
                  CHECK(status IN ('unread','reading','read','skip')),
    file_mtime    INTEGER NOT NULL DEFAULT 0,
    missing_since DATETIME,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    type          TEXT NOT NULL DEFAULT '',
    abstract      TEXT NOT NULL DEFAULT '',
    language      TEXT NOT NULL DEFAULT '',
    author        TEXT NOT NULL DEFAULT '[]',
    translator    TEXT NOT NULL DEFAULT '[]',
    origtitle     TEXT NOT NULL DEFAULT '',
    edition       TEXT NOT NULL DEFAULT '',
    volume        TEXT NOT NULL DEFAULT '',
    series        TEXT NOT NULL DEFAULT '',
    series_number TEXT NOT NULL DEFAULT '',
    publisher     TEXT NOT NULL DEFAULT '',
    year          TEXT NOT NULL DEFAULT '',
    note          TEXT NOT NULL DEFAULT '',
    keywords      TEXT NOT NULL DEFAULT '[]',
    isbn          TEXT NOT NULL DEFAULT '',
    links         TEXT NOT NULL DEFAULT '[]'
);

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

CREATE TABLE IF NOT EXISTS page_notes (
    page_id    INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    body       TEXT    NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS page_drawings (
    page_id    INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    svg        TEXT    NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS page_status (
    page_id    INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    status     TEXT    NOT NULL DEFAULT 'unread'
               CHECK(status IN ('unread','reading','read','skip')),
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS page_ocr (
    page_id    INTEGER PRIMARY KEY REFERENCES pages(id) ON DELETE CASCADE,
    body       TEXT    NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ── Sections ───────────────────────────────────────────────────

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

-- ── Collections ────────────────────────────────────────────────

-- library_id is a legacy column kept for schema compatibility.
-- Library membership is managed via library_collection_members.
CREATE TABLE IF NOT EXISTS book_collections (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    color       TEXT NOT NULL DEFAULT '#888888',
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS page_collections (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    color       TEXT NOT NULL DEFAULT '#888888',
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- ── Collection members ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS book_collection_members (
    collection_id TEXT NOT NULL REFERENCES book_collections(id) ON DELETE CASCADE,
    book_id       TEXT NOT NULL REFERENCES books(id),
    PRIMARY KEY(collection_id, book_id)
);

CREATE TABLE IF NOT EXISTS page_collection_members (
    collection_id TEXT    NOT NULL REFERENCES page_collections(id) ON DELETE CASCADE,
    page_id       INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    PRIMARY KEY(collection_id, page_id)
);

-- ── Library-collection membership (many-to-many) ───────────────

CREATE TABLE IF NOT EXISTS library_collection_members (
    library_id    TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    collection_id TEXT NOT NULL REFERENCES book_collections(id) ON DELETE CASCADE,
    PRIMARY KEY(library_id, collection_id)
);

-- ── Indexes ────────────────────────────────────────────────────

CREATE INDEX IF NOT EXISTS idx_pages_book                        ON pages(book_id);
CREATE INDEX IF NOT EXISTS idx_sections_book                     ON sections(book_id);
CREATE INDEX IF NOT EXISTS idx_book_collection_members_book      ON book_collection_members(book_id);
CREATE INDEX IF NOT EXISTS idx_page_collection_members_page      ON page_collection_members(page_id);
CREATE INDEX IF NOT EXISTS idx_library_collection_members_coll   ON library_collection_members(collection_id);
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

	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return &Store{db: db}, nil
}

func runMigrations(db *sql.DB) error {
	// Seed Central Library with a fixed UUID so it always exists.
	if _, err := db.Exec(
		`INSERT OR IGNORE INTO libraries (id, name) VALUES (?, 'Central Library')`,
		centralLibraryUUID,
	); err != nil {
		return fmt.Errorf("seed central library: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
