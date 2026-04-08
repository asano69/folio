# Openbook
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
- go標準モジュールのflagで、openbookコマンドを作り、openbook serverでサーバが起動するようにする。
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
openbook
├── cmd
│   └── server
│       └── main.go
├── go.mod
├── internal
│   ├── config
│   │   └── config.go
│   ├── handlers
│   │   ├── books.go
│   │   ├── images.go
│   │   └── viewer.go
│   └── server
│       └── server.go
├── src
│   ├── main.ts
│   ├── tsconfig.json
│   ├── ui
│   │   └── components.ts
│   └── viewer
│       ├── display.ts
│       └── navigation.ts
├── static
│   ├── app.js
│   ├── app.js.map
│   └── style.css
├── templates
│   ├── books.html
│   ├── layout.html
│   └── viewer.html
```

TypeScriptの詳細構成:
```
src/
├── main.ts              # エントリポイント
├── viewer/
│   ├── navigation.ts    # ページナビゲーション
│   └── display.ts       # 画像表示制御
└── ui/
    └── components.ts    # UI部品・イベント処理
```
