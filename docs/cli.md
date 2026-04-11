
## CLI Reference

### Synopsis
```
folio <subcommand> [arguments]
```

Configuration is read exclusively from environment variables (see Configuration
section). The `folio.env` file is not loaded automatically; use
`env $(cat folio.env | xargs) folio <subcommand>` or the Air dev setup.

---

### Subcommands

#### `folio server`

Start the HTTP server.

Reads `FOLIO_HOST` and `FOLIO_PORT` for the bind address. Opens the SQLite
database at `FOLIO_DATA_PATH`, applies the schema and any pending migrations,
then registers all HTTP routes and begins serving.

**Exit**: non-zero on listen error or store open failure.

---

#### `folio scan [path]`
Scan CBZ files and synchronise the database.

| Argument | Default | Description |
|----------|---------|-------------|
| `path` | `FOLIO_LIBRARY_PATH` | Directory to scan (recursive) |

**Phase 1 — ScanMeta**: reads only `folio.json` from each CBZ to obtain the
book UUID and file modification time. No image data is read at this stage.

**Phase 2 — OpenBook**: runs only for books whose UUID is absent or whose
`file_mtime` differs from the stored value. Generates a UUID and writes
`folio.json` when the file is missing. Lists all image entries and computes a
SHA-256 hash per image.

**Database updates**:
- `UpsertBook` for every found book — updates `title`, `source`, `file_mtime`,
  and clears `missing_since`. User-set `status` is preserved.
- `UpsertImages` for changed books — deletes and re-inserts `pages` rows, then
  rebuilds the `sections` table from `notes` where `attribute = 'section'`.
- `GenerateThumbnail` + `UpsertThumbnail` for books that have no stored
  thumbnail. Thumbnail generation is parallelised; DB writes are sequential.
- `MarkBookMissing` for books in the DB whose CBZ was not found under `path`.
  Sets `missing_since` only when it is currently NULL, preserving the original
  disappearance timestamp across repeated scans.

**Partial scan**: when `path` is a subdirectory, missing-book detection is
restricted to books whose `source` path starts with that directory. Books
outside the scanned subtree are not marked missing.

**Stdout**: scan progress and a final summary line.
**Stderr**: per-file errors for skipped CBZs; skipped thumbnail warnings.
**Exit**: non-zero on store open or scan walk failure.

---

#### `folio thumbnail <uuid>`

Regenerate the book-level thumbnail for a single book.
| Argument | Description |
|----------|-------------|
| `uuid` | Book UUID as stored in `folio.json` |

Opens the CBZ, decodes the first image, scales it to 400 px wide (JPEG,
quality 85), and writes the result to `thumbnails` via `UpsertThumbnail`.

**Exit**: non-zero if the book is not found, the CBZ cannot be opened, or the
DB write fails.

---

#### `folio page-thumbnails [uuid]`

Generate page-level thumbnails for one book or the entire library.

| Argument | Default | Description |
|----------|---------|-------------|
| `uuid` | *(omit)* | Limit to a single book; omit to process all non-missing books |

For each book, lists images from the DB and skips any that already have a row
in `page_thumbnails` or have an empty hash (i.e. `folio hash` has not been run
yet). Batches remaining images by book so each CBZ is opened exactly once.
Thumbnail generation (300 px wide, JPEG quality 85) is parallelised across
`GOMAXPROCS` workers; DB writes are sequential.

**Stdout**: count of generated thumbnails.
**Stderr**: per-book errors and per-image skip warnings (missing hash).
**Exit**: non-zero on store open failure or DB write error.

---

#### `folio hash <uuid>`

Recompute image hashes for a single book.

| Argument | Description |
|----------|-------------|
| `uuid` | Book UUID |

Runs a full `OpenBook` on the CBZ (lists images and computes SHA-256 per
image), then calls `UpsertImages` to refresh `pages` rows and rebuild
`sections`. Use this after manually modifying a CBZ's image contents when the
file modification time alone is not sufficient to trigger a re-hash during
`folio scan`.

**Stdout**: confirmation with image count.
**Exit**: non-zero if the book is not found or the CBZ cannot be opened.

---

### Typical Workflow

```bash
# Initial library registration
folio scan

# Generate book cover thumbnails (done automatically by scan for new books)
# Regenerate a single cover after replacement:
folio thumbnail <uuid>

# Generate page-level thumbnails (not run automatically by scan)
folio page-thumbnails

# After manually editing a CBZ's images:
folio hash <uuid>
folio page-thumbnails <uuid>

# Start the server
folio server
```

