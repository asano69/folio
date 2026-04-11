## Why Use an Archive Format (CBZ)?

Folio stores materials in the CBZ format. This choice is based on technical efficiency, metadata locality, compatibility with existing assets, and alignment with the system’s data ownership philosophy.

### Technical Rationale

CBZ is a ZIP archive. By reading the ZIP central directory (located at the end of the file), the byte offsets of all entries can be determined without extracting the entire archive. This enables random access to individual pages using seek-and-decompress operations.

The `OpenPage` implementation in `internal/storage/cbz.go` follows this approach, allowing a single page to be accessed without loading the full archive into memory. This provides efficient page-level access while preserving a single-file representation.

### Metadata Co-location

By storing `folio.json` inside the ZIP archive, the UUID and title move together with the file. This ensures that the identifier is preserved even when files are moved between directories. The design described in *Data Ownership Philosophy* (`docs/design-02.md`) relies on this property.

The archive therefore acts as a self-contained unit containing both content and identity metadata.

### Compatibility with Existing Assets

Scanned books and technical materials are commonly distributed and managed in CBZ/CBR formats. Using CBZ allows existing collections to be imported without conversion, reducing friction and preserving prior organization.

### Simplicity

CBZ avoids the need to interpret complex binary formats such as PDF. The implementation relies only on the standard `archive/zip` library. Since a CBZ file is simply a collection of images, page order is determined by filename sorting (as implemented in `listImages`), and no additional abstraction layer is required.

In practice, the primary technical advantages are:

* random access via ZIP central directory
* simple metadata embedding
* minimal implementation complexity

### Comparison with Storing Images Directly in the Filesystem

Storing pages as individual image files results in hundreds of files per book. This increases filesystem entry counts and introduces operational overhead, including:

* inode consumption
* directory listing and traversal cost (`ls`, `WalkDir`)
* increased file count during backup and transfer

Using CBZ preserves a one-book-per-file model, which aligns with the conceptual structure of a bookshelf and simplifies file management.

### Comparison with Object Storage

Object storage assumes network-based API access, introducing latency and operational cost. Folio is designed for self-hosting and serving materials from local server storage, as stated in `README.md`. The system therefore does not assume a network-dependent storage infrastructure.

CBZ enables direct local filesystem access without requiring an external storage service.

### Conceptual Rationale

CBZ represents a single book as a single file. This provides several properties:

* copying, moving, and backing up require a single file operation
* metadata (`folio.json`) is physically bound to the book
* files remain directly manageable through standard file managers

In contrast, object storage or directories of raw images represent only a collection of pages. The concept of a “book” must then be reconstructed through an external database or directory structure.

Using CBZ allows the book to remain self-contained even if the database is lost. This property is fundamental to the Data Ownership Philosophy and ensures that books remain intact independent of the application.
