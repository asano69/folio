## Project Structure
```
folio/
├── cmd/folio/
│   ├── main.go        # CLI entry point; subcommand dispatch
│   └── server.go      # HTTP server setup and route registration
├── internal/
│   ├── config/
│   │   └── config.go  # Environment variable loading; Config struct
│   ├── handlers/
│   │   ├── home.go              # GET /
│   │   ├── collection_page.go   # GET /collections/{id}
│   │   ├── book_pages.go        # GET /books/{uuid}/overview|bibliography|pages/{num}
│   │   ├── images.go            # GET /images/{bookID}/{filename}
│   │   ├── book_thumbnail.go    # GET /thumbnails/{bookID}
│   │   ├── page_thumbnail.go    # GET /page-thumbnails/{bookID}/{pageHash}
│   │   ├── books_api.go         # /api/books/
│   │   ├── pages_api.go         # /api/pages/
│   │   └── collections_api.go   # /api/collections/
│   ├── storage/
│   │   ├── types.go      # Book and ImageEntry structs
│   │   ├── cbz.go        # CBZ open, folio.json read/write, image listing, page serving
│   │   ├── scan.go       # Recursive library walk; Scan and ScanMeta with worker pool
│   │   └── thumbnail.go  # Book-level and page-level JPEG thumbnail generation
│   └── store/
│       ├── store.go    # SQLite open, schema init, migration application
│       └── queries.go  # All DB read/write operations; domain type definitions
├── src/                        # TypeScript and CSS source
│   ├── main.ts                 # DOMContentLoaded init dispatcher
│   ├── api.ts                  # Centralized fetch helpers for all REST endpoints
│   ├── types.ts                # Shared frontend domain types (ReadStatus, NotePayload, etc.)
│   ├── viewer/
│   │   ├── navigation.ts  # Keyboard nav, page-jump selector
│   │   ├── display.ts     # Wheel zoom, drag-to-pan, double-click reset
│   │   ├── editor.ts      # Edit pane open/close, note save, snapshot/restore
│   │   ├── toc.ts         # TOC pane toggle
│   │   └── drawing.ts     # SVG drawing overlay, pen/eraser, undo/redo, save
│   ├── ui/
│   │   ├── search.ts        # Title filter for book grids
│   │   ├── rename.ts        # Inline book title rename in edit mode
│   │   ├── collections.ts   # Sidebar drag-drop, multi-select, create/rename/delete
│   │   ├── page-status.ts   # Per-page read status buttons
│   │   ├── book-note.ts     # Book-level memo save
│   │   └── components.ts    # Stub for future toast/modal UI elements
│   ├── css/
│   │   ├── base.css      # Design tokens (CSS variables), reset, site header
│   │   ├── pane.css      # Shared slide-in pane structure (TOC and Edit panes)
│   │   ├── shelf.css     # Library grid, book cards, search bar, missing books
│   │   ├── sidebar.css   # Collection sidebar, drag-over states, multi-select
│   │   ├── viewer.css    # Viewer layout, image display, note display, jump overlay
│   │   ├── toc.css       # TOC pane content styles
│   │   ├── editor.css    # Edit pane form styles
│   │   ├── book.css      # Per-book page grid and page card styles
│   │   ├── drawing.css   # SVG overlay, draw pane, tool buttons
│   │   └── overview.css  # Overview page tab nav, status tints, bibliographic layout
│   ├── style.css         # CSS entry point; imports all css/* files
│   └── folio.svg         # Application icon source
├── templates/
│   ├── layout.html        # Base HTML shell; defines title and content blocks
│   ├── sidebar.html       # Collection sidebar partial; included by home and collection templates
│   ├── home.html          # All-books library page
│   ├── collection.html    # Single collection book list
│   ├── overview.html      # Per-book page grid with status buttons
│   ├── bibliography.html # Per-book TOC, stats, and book-level memo
│   └── viewer.html        # Single-page viewer with TOC, edit, and draw panes
├── docs/
│   ├── design-01.md  # Initial design (superseded)
│   ├── design-02.md  # Data ownership philosophy (superseded)
│   ├── design-03.md  # Phase 3 schema (superseded)
│   └── design-04.md  # Current design reference (this document)
├── static/           # Build output (gitignored except favicon.ico)
├── Makefile          # Build, watch, docker, icon, clean targets
├── go.mod / go.sum   # Go module definition and checksums
├── shell.nix         # Nix development shell
├── .air.toml         # Air live-reload configuration
├── folio.env         # Local environment variable defaults
└── .gitignore
```