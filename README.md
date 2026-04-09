# folio
- サーバのファイルシステムに保存されたデジタル画像資料をベースにしたページ管理・注釈システムを実装する。 各ページに対してメタデータ・注釈・ナビゲーション構造を追加できるようにする。
- デジタル画像資料自体は変更せず、メタデータのみを管理する軽量なデジタル書籍ビューア兼ノートシステムとする。

## Tech Stack
- Backend: Go
- Template Engine: html/template (go-template)
- Frontend: Server Side Rendering
- TypeScript: 必要最小限（描画・UI操作のみ）
- Storage: ローカルファイルベース
- Database: Sqlite3

## 開発戦略
- go標準モジュールのflagで、folioコマンドを作り、folio serverでサーバが起動するようにする。
- サーバの設定は環境変数を使い宣言的に行うことができるようにする。

## 実装する機能
- ローカルに存在するスキャン画像にURIを与える機能
- セクションを登録することで簡単に、セクション間のページ移動をすることができる機能
- 各ページに、落書きとマークダウンノートを作成することができる機能
- 各ページに、タグをつけることができる機能
- 複数ページのコレクションを作成することができる機能

## プロジェクト構造
実装する構造:
```
folio
├── cmd/folio/
│   ├── main.go        # エントリポイント。server / scan / thumbnail <uuid> サブコマンド
│   └── server.go      # サーバ初期化・ルーティング
├── internal/
│   ├── config/
│   │   └── config.go  # 環境変数ロード
│   ├── storage/       # ファイルシステム・CBZ操作（DBの知識なし）
│   │   ├── types.go   # Book / Page 型
│   │   ├── cbz.go     # CBZ open / scan / folio.json read-write
│   │   ├── scan.go    # ライブラリルートを再帰walk
│   │   └── thumbnail.go # CBZからサムネイルJPEG生成
│   ├── store/         # SQLite操作（ファイルシステムの知識なし）
│   │   ├── store.go   # DB初期化・スキーマ定義
│   │   └── queries.go # books / pages / thumbnails テーブルのCRUD
│   └── handlers/      # HTTPハンドラ（storageとstoreを組み合わせる）
│       ├── books.go   # GET /
│       ├── viewer.go  # GET /viewer
│       ├── images.go  # GET /images/{bookID}/{filename}
│       ├── thumbnail.go # GET /thumbnails/{bookID}
│       └── api.go     # PUT /api/books/{id}
│                      # POST /api/books/{id}/thumbnail
├── templates/
│   ├── layout.html
│   ├── books.html
│   └── viewer.html
├── static/
│   ├── style.css
│   ├── app.js
│   └── app.js.map
├── src/               # TypeScriptソース
│   ├── main.ts
│   ├── viewer/
│   │   ├── navigation.ts
│   │   └── display.ts
│   └── ui/
│       ├── components.ts
│       └── rename.ts
├── go.mod
├── Makefile
└── folio.env
```

