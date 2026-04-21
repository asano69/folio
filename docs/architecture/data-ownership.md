## Folio Data Ownership Philosophy


---

**Data ownership**:
- `folio.json` inside the CBZ — immutable, book-intrinsic data (`id`, `title`).
  Writing here requires rewriting the entire ZIP, so only rarely-changing data
  belongs here.
- SQLite DB — mutable, user-subjective data and derived caches.

--- 

Folio divides data into two categories—**book-intrinsic data** and **user-generated data**—and stores them separately.

* **folio.json** — immutable, book-intrinsic data
* **SQLite database** — mutable, user-subjective data and derived caches

## Data Stored in `folio.json`

Only the following fields are stored in `folio.json`: **id, title, author, publication year, and publisher**. These constitute facts describing what the book is, and they do not change regardless of how the book is used.

The UUID, in particular, represents the identity of the book itself and must be physically embedded within the file. The guarantee that the identifier is preserved even if the file is moved is achieved only by storing this UUID inside the CBZ archive.

Writing to `folio.json` requires rewriting the entire ZIP archive (`cbz.go: writeMeta`). This operation is expensive, therefore frequently changing data cannot be stored in `folio.json`. Title updates (`UpdateTitle`) modify both `folio.json` and the database, but all other updates are written only to the database. This constraint keeps the design simple.

## Data Stored in SQLite

SQLite stores data actively created by the user, such as:

* notes
* tags
* collections
* page_status

These represent subjective records of how the book was read or organized and are independent of the book’s intrinsic content.

Running `folio scan` allows most of the database to be reconstructed as long as the CBZ files exist. However, notes and tags are manually created by the user and cannot be restored through scanning. This creates an asymmetry: only the database needs to be backed up to preserve user-generated data.

## Architectural Implication

This design asserts that **books exist outside the application**.

Conventional applications centralize data in a database; if the application disappears, the data is lost. Folio reverses this relationship: CBZ files are the primary entities, and the database is merely a layer of cache and annotations on top of them.

Even if the application is discarded, the CBZ files remain and can be opened in other viewers. Only user-created notes depend on Folio.

This is consistent with the README statement that "the digital image materials themselves are not modified; only metadata is managed," and provides the technical foundation for Folio’s goal of **preserving a sense of continuity and safety**.

