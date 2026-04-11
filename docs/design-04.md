# Folio Design Document 04

This document supersedes design-01.md through design-03.md where they conflict.
It reflects the current state of the codebase.

---

## URL Scheme

### HTML Pages

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/` | `HomeHandler` | All-books library page |
| GET | `/collections/{id}` | `CollectionPageHandler` | Single collection book list |
| GET | `/books/{uuid}/overview` | `BookDispatchHandler` | Page grid with status and thumbnails |
| GET | `/books/{uuid}/bibliography` | `BookDispatchHandler` | TOC, stats, and book-level memo |
| GET | `/books/{uuid}/pages/{num}` | `BookDispatchHandler` | Single-page viewer with edit and draw panes |

### Static Assets and Media

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/static/{file}` | `http.FileServer` | JS, CSS, favicon |
| GET | `/images/{bookID}/{filename}` | `ImageHandler` | Raw image served directly from CBZ |
| GET | `/thumbnails/{bookID}` | `BookThumbnailHandler` | Book-level JPEG thumbnail from DB |
| GET | `/page-thumbnails/{bookID}/{pageHash}` | `PageThumbnailHandler` | Page-level JPEG thumbnail from DB |

### REST API

#### Books — `/api/books/`

| Method | Path | Description |
|--------|------|-------------|
| PUT | `/api/books/{id}` | Rename a book (updates both folio.json and DB) |
| PUT | `/api/books/{id}/note` | Save book-level memo |
| POST | `/api/books/{id}/thumbnail` | Regenerate book thumbnail |

#### Pages — `/api/pages/`

| Method | Path | Description |
|--------|------|-------------|
| PUT | `/api/pages/{bookID}/{pageHash}` | Save text note (title, attribute, body) |
| PUT | `/api/pages/{bookID}/{pageHash}/drawing` | Save or clear SVG drawing |
| PUT | `/api/pages/{bookID}/{pageHash}/status` | Update read status |

The note and drawing endpoints are intentionally separate: saving text never
overwrites an existing drawing, and saving a drawing never touches text fields.

#### Collections — `/api/collections/`

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/collections/` | Create a collection |
| PUT | `/api/collections/{id}` | Rename a collection |
| DELETE | `/api/collections/{id}` | Delete a collection and all its memberships |
| POST | `/api/collections/{id}/books/{bookID}` | Add a book to a collection |
| DELETE | `/api/collections/{id}/books/{bookID}` | Remove a book from a collection |

### Handler File Map

| File | Struct | Responsibility |
|------|--------|----------------|
| `home.go` | `HomeHandler` | `GET /` |
| `collection_page.go` | `CollectionPageHandler` | `GET /collections/{id}` |
| `book_pages.go` | `BookDispatchHandler` | `GET /books/{uuid}/...` |
| `images.go` | `ImageHandler` | `GET /images/...` |
| `book_thumbnail.go` | `BookThumbnailHandler` | `GET /thumbnails/...` |
| `page_thumbnail.go` | `PageThumbnailHandler` | `GET /page-thumbnails/...` |
| `books_api.go` | `BooksAPIHandler` | `/api/books/` |
| `pages_api.go` | `PagesAPIHandler` | `/api/pages/` |
| `collections_api.go` | `CollectionsAPIHandler` | `/api/collections/` |

---

## Database Schema

SQLite. The database file is at `${FOLIO_DATA_PATH}/folio.db`.

### Design Principles

**Re-scan safety**: `UpsertPages` deletes and re-inserts all page rows, so
`pages(id)` changes on every scan. Any table holding per-page user data is
keyed by `(book_id, page_hash)` — not by `pages(id)` — so data survives a
re-scan. This applies to: `notes`, `page_thumbnails`, `page_status`,
`page_ocr`, `page_tags`.

**Data ownership**:
- `folio.json` inside the CBZ — immutable, book-intrinsic data (`id`, `title`).
  Writing here requires rewriting the entire ZIP, so only rarely-changing data
  belongs here.
- SQLite DB — mutable, user-subjective data and derived caches.

### Tables

```sql
CREATE TABLE IF NOT EXISTS books (
    id            TEXT PRIMARY KEY,            -- UUID from folio.json
    title         TEXT NOT NULL,
    source        TEXT NOT NULL,               -- absolute path to CBZ
    status        TEXT NOT NULL DEFAULT 'unread'
                  CHECK(status IN ('unread','reading','read','skip')),
    file_mtime    INTEGER NOT NULL DEFAULT 0,  -- Unix timestamp; detects CBZ changes
    missing_since DATETIME,                    -- set when CBZ not found on last scan
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pages (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id  TEXT    NOT NULL REFERENCES books(id),
    number   INTEGER NOT NULL,
    filename TEXT    NOT NULL,                 -- entry name inside CBZ
    hash     TEXT    NOT NULL DEFAULT '',      -- SHA-256 of uncompressed image bytes
    UNIQUE(book_id, number)
);

-- Book-level JPEG thumbnail.
CREATE TABLE IF NOT EXISTS thumbnails (
    book_id    TEXT PRIMARY KEY REFERENCES books(id),
    data       BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Page-level JPEG thumbnail. Keyed by page_hash for re-scan safety.
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

-- Per-page read status. Keyed by page_hash for re-scan safety.
CREATE TABLE IF NOT EXISTS page_status (
    book_id    TEXT NOT NULL REFERENCES books(id),
    page_hash  TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'unread'
               CHECK(status IN ('unread','reading','read','skip')),
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(book_id, page_hash)
);

-- OCR text per page. Keyed by page_hash for re-scan safety.
CREATE TABLE IF NOT EXISTS page_ocr (
    book_id    TEXT NOT NULL REFERENCES books(id),
    page_hash  TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY(book_id, page_hash)
);

-- Sections derived from notes where attribute = 'section'.
-- end_page is not stored; derived as next section's start_page - 1.
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

-- Book-level memo. One row per book.
CREATE TABLE IF NOT EXISTS book_notes (
    book_id    TEXT PRIMARY KEY REFERENCES books(id),
    body       TEXT NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Collection-level memo. One row per collection.
CREATE TABLE IF NOT EXISTS collection_notes (
    collection_id INTEGER PRIMARY KEY REFERENCES collections(id),
    body          TEXT NOT NULL DEFAULT '',
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tags scoped per entity type; not shared across scopes.
-- color is a CSS hex string e.g. '#ff0000'.
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
```

### Indexes

```sql
CREATE INDEX IF NOT EXISTS idx_book_tags_tag       ON book_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_page_tags_tag       ON page_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_note_tags_tag       ON note_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_collection_tags_tag ON collection_tags(tag_id);
CREATE INDEX IF NOT EXISTS idx_sections_book       ON sections(book_id);
```

### Table Reference

| Table | Key | Recoverable by scan? | Notes |
|-------|-----|----------------------|-------|
| `books` | `id` (UUID) | ✅ | `file_mtime` used to skip unchanged CBZs |
| `pages` | `(book_id, number)` | ✅ | Integer PK changes on re-scan — never FK from user data |
| `thumbnails` | `book_id` | ✅ | Book-level JPEG |
| `page_thumbnails` | `(book_id, page_hash)` | ✅ | Page-level JPEG |
| `notes` | `(book_id, page_hash)` | ❌ | Has integer PK for `note_tags`; holds `svg_drawing` |
| `page_status` | `(book_id, page_hash)` | ❌ | |
| `page_ocr` | `(book_id, page_hash)` | ✅ | Derived from image content |
| `sections` | `(book_id, start_page)` | ❌ | Synced from `notes.attribute = 'section'`; has own `status` |
| `book_notes` | `book_id` | ❌ | One memo per book |
| `collection_notes` | `collection_id` | ❌ | One memo per collection |
| `tags` | `(scope, name)` | ❌ | Scoped to book / page / note / collection |
| `book_tags` | `(book_id, tag_id)` | ❌ | |
| `page_tags` | `(book_id, page_hash, tag_id)` | ❌ | Re-scan safe |
| `note_tags` | `(note_id, tag_id)` | ❌ | Uses `notes` integer PK |
| `collection_tags` | `(collection_id, tag_id)` | ❌ | |
| `collections` | `id` | ❌ | |
| `collection_books` | `(collection_id, book_id)` | ❌ | |

### Status Values

Used on `books.status`, `page_status.status`, and `sections.status`:

| Value | Meaning |
|-------|---------|
| `unread` | Not yet read |
| `reading` | Currently in progress |
| `read` | Finished |
| `skip` | Intentionally skipped |

### Page Attributes

Stored as plain strings in `notes.attribute`. No DB CHECK constraint so the
list can evolve without a schema migration.

| Value | Meaning |
|-------|---------|
| `cover` | Cover page |
| `toc` | Table of contents page |
| `section` | Section start — drives the `sections` table and viewer TOC |
| `page` | Ordinary content page |
| `index` | Index page |
| `other` | Anything else |

Setting a page's attribute to `section` automatically upserts a row in the
`sections` table. Removing or changing the attribute deletes it.

---

## Scan Logic

```
folio scan [path]
  │
  ├─ Phase 1 (ScanMeta): read only folio.json from each CBZ
  │    ├─ ID present + mtime unchanged → skip full open (unchanged)
  │    └─ ID missing or mtime changed  → queue for full open
  │
  ├─ Phase 2 (OpenBook): full open for new/changed CBZs only
  │    ├─ folio.json missing → generate UUID, write folio.json into CBZ
  │    └─ list images, compute SHA-256 hash per image
  │
  ├─ UpsertBook for all found books (clears missing_since)
  ├─ UpsertImages for changed books (rebuilds pages + sections tables)
  ├─ GenerateThumbnail for books without a stored thumbnail
  └─ MarkBookMissing for books in DB not found on disk
```

`folio scan` can be run on a subdirectory to partially scan the library.
Missing-book detection is restricted to books whose source path is under
the scanned directory, so books outside the scan path are not marked missing.

---

## Configuration

All configuration is via environment variables. Defaults match `folio.env`.

| Variable | Default | Description |
|----------|---------|-------------|
| `FOLIO_LIBRARY_PATH` | `./library` | Root directory for CBZ files |
| `FOLIO_DATA_PATH` | `./data` | Directory for SQLite database |
| `FOLIO_HOST` | `0.0.0.0` | Server bind address |
| `FOLIO_PORT` | `3000` | Server port |