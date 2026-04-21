
# Re-scan Safety

`UpsertPages` deletes and re-inserts all page rows, so `pages(id)` (integer PK) changes on every scan. Any table that holds per-page user data must therefore be keyed by `(book_id, page_hash)` — not by `pages(id)` — so that data survives a re-scan. This applies to: `notes`, `page_thumbnails`, `page_status`, `page_ocr`, `page_tags`.

