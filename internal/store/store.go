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
CREATE TABLE IF NOT EXISTS books (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    source      TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id     TEXT    NOT NULL REFERENCES books(id),
    number      INTEGER NOT NULL,
    filename    TEXT    NOT NULL,
    hash        TEXT    NOT NULL DEFAULT '',
    UNIQUE(book_id, number)
);

CREATE TABLE IF NOT EXISTS thumbnails (
    book_id     TEXT PRIMARY KEY REFERENCES books(id),
    data        BLOB NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Notes holds user-authored metadata for a single page.
-- The primary key uses (book_id, page_hash) instead of a FK to pages(id)
-- so that data survives a re-scan (which replaces all pages rows) and is
-- also stable when pages are deleted from the CBZ (which shifts page numbers).
CREATE TABLE IF NOT EXISTS notes (
    book_id     TEXT NOT NULL REFERENCES books(id),
    page_hash   TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    attribute   TEXT NOT NULL DEFAULT '',
    body        TEXT NOT NULL DEFAULT '',
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (book_id, page_hash)
);
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

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
