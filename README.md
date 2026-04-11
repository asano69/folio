# Folio

- サーバのファイルシステムに保存されたデジタル画像資料をベースにしたページ管理・注釈システム。
- 各ページに対してメタデータ・注釈・ナビゲーション構造を追加できるようにする。
- デジタル画像資料自体は変更せず、メタデータのみを管理する軽量なデジタル書籍ビューア兼ノートシステムとする。

## 目的
### 問題
- スキャンしたデジタル資料を有効活用できていない。
- 既存のOSS書籍ビューアは、漫画のコレクションを作成し表示するなどの基本的な機能しかないものがほとんどでメモ・ノート・マーカ・タグ付けなどの機能が十分に統合されていない。
- また、ビューワはSPAとして作られているものが多い。SPAだと、すべてのページに静的なURIが与えられていないように感じられ、特定のページのURLを別のノートシステムから参照するような使い方ができない。

### 目標
- スキャンした資料に、それぞれURIを付与して、リソースへの永続的なアクセスを保証する。
- スキャンした資料にコメントをつけていつでも参照できるように使いやすくしたい。
- 素早く内容を理解するために、書き込みをしたい。

### 非目標
- 二人以上の人と共有して使用することはこのアプリの目標ではない。
- アプリを世界に一般公開することは想定していない。そもそも、スキャンした書籍を一般公開することは絶対にしてはいけないし、スキャンすること自体が違法な国もある。

### 要件
- 本棚に物理的に配架された本に内在する、安心感、存在感、信頼感をできる限り損ねないようにする
	- 本棚に並べられた本はいつアクセスしても変わらないため、そこにある情報は失われないという安心感がある。
- 物理的な本に内在するコンテンツへのアクセス性を上回ること
	- ブラウジング：　本はパラパラとめくることで情報の所在の概観をつかむことができる。
	- 目次：　情報の構造の外観を掴むことができる。
	- 付箋：　ページのタグ
	- 書き込み：　記憶を思い出すきっかけになる。
- ページ番号：
- 切り抜き可能：
	- 切り抜きして他のノートに貼れるような機能を提供する。

## 機能
- ローカルに存在するスキャン画像にURIを与える機能
- セクションを登録することで簡単に、セクション間のページ移動をすることができる機能
- 各ページに、落書きとマークダウンノートを作成することができる機能
- 各ページに、タグをつけることができる機能
- 複数ページのコレクションを作成することができる機能
- **ページにURLを振る**: スキャンした資料のすべてのページを画像にしそれぞれにURLをふることで容易に参照できるようにする。
- **ページに注釈を残す**: ページの要約、タグ、コメント、ペンによる書き込みを残すことができる。
    - **要約**: 
    - **タグ**: 
    - **コメント**: 
    - **落書き**: 
- **目次**: 資料に目次を作成して、目次を利用して任意の場所から読み始めることができる。
- **進捗率**: 資料のどこを読んだか、どこを読んでいないかを区別することができる。


## 設計
### 技術
- Backend: Go
- Template Engine: html/template (go-template)
- Frontend: Server Side Rendering
- TypeScript: 必要最小限（描画・UI操作のみ）
- Storage: ローカルファイルベース
- Database: Sqlite3

### CLI Program
- そもそも、サーバのローカルストレージに存在する資料を配信するセルフホスティングアプリなので、Shellが使えることが前提のため、わざわざGUIを作らなくても十分な場面も多い。
- CLIを用意することで、フロントエンドの管理用GUIが完成していない段階からアプリを使い始めることができる。
- goのflagで、folioコマンドを作り、その中のserverサブコマンドでサーバを起動するようにする。サーバの設定は環境変数を使い変更することができるようにする。

## プロジェクト構造
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
│   ├── bibliographic.html # Per-book TOC, stats, and book-level memo
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





