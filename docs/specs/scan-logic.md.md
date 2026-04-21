# Scan Logic

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
