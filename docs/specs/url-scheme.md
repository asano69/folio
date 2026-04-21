# URL Scheme

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


