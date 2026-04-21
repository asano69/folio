# Folio Design Document 02

## Data Ownership Philosophy

Folio uses two storage layers. The rule for deciding where data lives is simple:

- **folio.json** — immutable, book-intrinsic data
- **SQLite DB** — mutable, user-subjective data and derived caches

---

## folio.json (inside CBZ)

Holds data that is intrinsic to the book itself and does not change based on the user's perspective.

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Book A"
}
```

| Field | Reason |
|-------|--------|
| `id` | Identity must travel with the file |
| `title` | Intrinsic to the book, not the user |

Writing to folio.json requires rewriting the entire CBZ (ZIP structure constraint),
so only data that changes rarely belongs here.

---

## SQLite DB

Holds data that is subjective, mutable, or derivable.

| Table | Data | Recoverable? |
|-------|------|--------------|
| `books` | id, title, source path | ✅ `folio scan` |
| `pages` | filename, order | ✅ `folio scan` |
| `thumbnails` | JPEG blob | ✅ `folio scan` |
| `sections` | user-defined chapter structure | ❌ backup required |
| `notes` | per-page markdown | ❌ backup required |
| `tags` | user-defined labels | ❌ backup required |

Tags, notes, and sections reflect the user's subjective view of the material
and have no place in the book file itself.

---

## DB Reconstruction

`folio scan` can fully reconstruct the `books`, `pages`, and `thumbnails` tables
from the CBZ files and their embedded folio.json.

User-created data (`notes`, `sections`, `tags`) exists only in the DB.
**Back up the DB file to avoid losing this data.**

---

## Rename Consistency

Renaming a book updates both folio.json inside the CBZ and the DB title column.
This keeps the two layers consistent and means the correct title is recovered
after a `folio scan` even if the DB is lost.