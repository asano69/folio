# Folio Design Document

## Overview

A lightweight digital book viewer and note system.
Manages metadata only; the image files themselves are never modified.

---

## Tech Stack

- **Backend**: Go (standard library)
- **Frontend**: Server-side rendering with `html/template`
- **TypeScript**: Minimal, UI interaction only
- **Database**: SQLite
- **Storage**: Local filesystem (CBZ files)

---

## Storage Strategy

### Library Layout

The library root is specified via the `FOLIO_LIBRARY_PATH` environment variable.
Subdirectories are purely for the user's own organization and carry no meaning to the application.
Scanning is recursive: all `.cbz` files found under the library root are registered.

```
library/
├── manga/
│   ├── book-a.cbz
│   └── book-b.cbz
├── technical/
│   └── book-c.cbz
└── book-d.cbz
```

### CBZ Format

CBZ is a ZIP archive. Page images and a `folio.json` file are stored at the root of the archive.
Using `folio.json` rather than a generic name like `metadata.json` avoids conflicts with other
CBZ tools (e.g. ComicRack uses `ComicInfo.xml`).

```
book-a.cbz (ZIP)
├── folio.json
├── 001.jpg
├── 002.jpg
└── 003.jpg
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Book A"
}
```

### Page Access

Because CBZ is a ZIP file, random access to individual pages is possible without loading the entire archive into memory.
ZIP's central directory (stored at the end of the file) maps each entry to its byte offset,
so a single page can be extracted with a seek + decompress of that entry alone.

---

## Book Identity

Book identity is managed by a UUID stored inside `folio.json` within the CBZ.
This decouples identity from file location, so moving a CBZ between subdirectories does not lose its metadata.

**Scan logic:**

```
Open CBZ
  ├─ folio.json exists → read UUID
  └─ folio.json missing → generate UUID, write folio.json into CBZ

UUID exists in DB → update source path only (file may have moved)
UUID missing in DB → insert new record
```

---

## Database

SQLite. The database path is specified via the `FOLIO_DATA_PATH` environment variable.

### Phase 1 (implement now)

```sql
CREATE TABLE books (
    id          TEXT PRIMARY KEY,   -- UUID from folio.json
    title       TEXT NOT NULL,
    source      TEXT NOT NULL,      -- path relative to library root
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id     TEXT NOT NULL REFERENCES books(id),
    number      INTEGER NOT NULL,
    filename    TEXT NOT NULL,      -- entry name inside CBZ
    UNIQUE(book_id, number)
);
```

### Phase 2 (next)

```sql
CREATE TABLE sections (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id     TEXT NOT NULL REFERENCES books(id),
    title       TEXT NOT NULL,
    start_page  INTEGER NOT NULL,
    end_page    INTEGER             -- NULL means until next section
);

CREATE TABLE notes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id     INTEGER NOT NULL UNIQUE REFERENCES pages(id),
    markdown    TEXT NOT NULL DEFAULT '',
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Phase 3 (later)

```sql
CREATE TABLE tags (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    name    TEXT NOT NULL UNIQUE
);

CREATE TABLE page_tags (
    page_id INTEGER NOT NULL REFERENCES pages(id),
    tag_id  INTEGER NOT NULL REFERENCES tags(id),
    PRIMARY KEY(page_id, tag_id)
);

CREATE TABLE collections (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    title       TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE collection_pages (
    collection_id   INTEGER NOT NULL REFERENCES collections(id),
    page_id         INTEGER NOT NULL REFERENCES pages(id),
    position        INTEGER NOT NULL,
    PRIMARY KEY(collection_id, page_id)
);
```

---

## Internal Package Structure

```
internal/
├── config/     # environment variable loading
├── storage/    # CBZ open, page read, scan
├── store/      # SQLite read/write
├── handlers/   # HTTP handlers
└── server/     # routing
```

**Responsibilities:**

| Package | Role |
|---------|------|
| `storage` | Filesystem scan, CBZ open/read, UUID write |
| `store` | All SQLite operations |
| `handlers` | HTTP only; calls storage and store |

The `storage` package owns no DB knowledge. The `store` package owns no filesystem knowledge.

### Storage Interface

Defined now to allow future substitution (e.g. object storage) without changing handlers:

```go
type Backend interface {
    Open(path string) (io.ReadCloser, error)
    List(prefix string) ([]string, error)
}
```

---

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `FOLIO_LIBRARY_PATH` | `./library` | Root directory for CBZ files |
| `FOLIO_DATA_PATH` | `./data` | Directory for SQLite database |
| `FOLIO_HOST` | `0.0.0.0` | Server bind address |
| `FOLIO_PORT` | `3000` | Server port |

---

## Features Roadmap

| Priority | Feature |
|----------|---------|
| Now | CBZ scan, page viewer |
| Next | Sections, Markdown notes per page |
| Later | Tags per page, multi-page collections |
