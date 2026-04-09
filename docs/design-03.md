# Folio Design Document 03

## Schema (Phase 3)

This document supersedes design-01.md and design-02.md where they conflict.

---

## Key Design Principles

### Re-scan Safety

`UpsertPages` deletes and re-inserts all page rows, so `pages(id)` (integer PK)
changes on every scan. Any table that holds per-page user data must therefore be
keyed by `(book_id, page_hash)` — not by `pages(id)` — so that data survives a
re-scan. This applies to: `notes`, `page_thumbnails`, `page_status`, `page_ocr`,
`page_tags`.

### Data Ownership (unchanged from design-02)

- **folio.json** — immutable, book-intrinsic data (id, title)
- **SQLite DB** — mutable, user-subjective data and derived caches

---

## Full Schema

```sql
CREATE TABLE IF NOT EXISTS books (
    id            TEXT PRIMARY KEY,
    title         TEXT NOT NULL,
    source        TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'unread'
                  CHECK(status IN ('unread','reading','read','skip')),
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
-- Synced automatically when a page's attribute is set to 'section'.
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
```

---

## Table Reference

| Table | Key | Notes |
|-------|-----|-------|
| `books` | `id` (UUID from folio.json) | Added `status` column |
| `pages` | `(book_id, number)` | Integer PK changes on re-scan — do not FK from user data |
| `thumbnails` | `book_id` | Book-level JPEG thumbnail |
| `page_thumbnails` | `(book_id, page_hash)` | Page-level JPEG thumbnail |
| `notes` | `(book_id, page_hash)` | Has integer PK for note_tags; includes `svg_drawing` |
| `page_status` | `(book_id, page_hash)` | unread / reading / read / skip |
| `page_ocr` | `(book_id, page_hash)` | Raw OCR text |
| `sections` | `(book_id, start_page)` | Synced from `attribute = 'section'`; has own `status` |
| `book_notes` | `book_id` | One memo per book |
| `collection_notes` | `collection_id` | One memo per collection |
| `tags` | `(scope, name)` | Scoped to book / page / note / collection |
| `book_tags` | `(book_id, tag_id)` | |
| `page_tags` | `(book_id, page_hash, tag_id)` | Re-scan safe |
| `note_tags` | `(note_id, tag_id)` | Uses notes integer PK |
| `collection_tags` | `(collection_id, tag_id)` | |
| `collections` | `id` | |
| `collection_books` | `(collection_id, book_id)` | |

---

## Design Decisions

### Status values

Used on `books.status`, `page_status.status`, and `sections.status`:

| Value | Meaning |
|-------|---------|
| `unread` | Not yet read |
| `reading` | Currently in progress |
| `read` | Finished |
| `skip` | Intentionally skipped |

### Tags

Tags are **not shared** across entity types. A tag named "important" in scope
`page` is a different entity from one named "important" in scope `book`. This
keeps queries simple and avoids cross-entity coupling.

Each tag has a `color` (CSS hex string) and a `name`. No grouping.

### Sections

Sections are independent entities with their own `status`. They are derived from
pages whose `notes.attribute = 'section'`, but stored separately so that:

- Section-level read status can be tracked independently.
- Knowing which section a page belongs to can be computed from `sections.start_page`
  without scanning all notes.
- The derivation rule: when a note's attribute is set to or removed from `'section'`,
  the `sections` table should be updated (upsert / delete) to stay in sync.

`end_page` is not stored. It is derived dynamically as `next_section.start_page - 1`
(or the last page of the book for the final section).

### Notes and SVG

A page has at most one note row. The note holds both a text `body` and an
optional `svg_drawing` (raw SVG markup). The two coexist in the same row.
`svg_drawing` is NULL when no drawing has been saved.

### book_notes / collection_notes

One memo per book and one memo per collection. Simple text, no title field.
Separate from the `notes` table (which is page-scoped).
