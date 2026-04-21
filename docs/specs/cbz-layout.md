## Storage Strategy

### Library Layout

The library root is specified via the `FOLIO_LIBRARY_PATH` environment variable.
Subdirectories are purely for the user's own organization and carry no meaning to the application.
Scanning is recursive: all `.cbz` files found under the library root are registered.

```
library/
├── manga/
│   ├── book-a.cbz
│   └── book-b.cbz
├── technical/
│   └── book-c.cbz
└── book-d.cbz
```

### CBZ Format

CBZ is a ZIP archive. Page images and a `folio.json` file are stored at the root of the archive.
Using `folio.json` rather than a generic name like `metadata.json` avoids conflicts with other
CBZ tools (e.g. ComicRack uses `ComicInfo.xml`).

```
book-a.cbz (ZIP)
├── folio.json
├── 001.jpg
├── 002.jpg
└── 003.jpg
```

```json
{
  "version": "2026-04-20",
  "id": "019da294-f490-73d7-8dbd-a81da1aec5f2",
  "type": "book",
  "abstract": "",
  "language": "ja",
  "author": [
    { "family": "Yamada", "given": "Taro" }
  ],
   "translator": [
    { "family": "細谷", "given": "貞雄" }
  ],
  "title": "知識管理のためのノート術",
  "origtitle": "Note-taking for Knowledge Management",
  "edition": "2",
  "volume": "1",
  "series": "情報学ライブラリ",
  "series_number": "12",
  "publisher": "架空出版",
  "year": "2024",
  "note": "参考文献と索引あり",
  "keywords": ["知識管理", "ノート術", "情報整理"],
  "isbn": "",
  "links": ["https://example.org/books/asano2026-example"],
  "created_at": "2026-04-18T07:12:20+09:00",
  "updated_at": "2026-04-20T00:34:42+09:00"
}
```

