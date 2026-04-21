# Folio

<img src="src/icons/folio.svg" width="90" align="right" />

Folioはローカルサーバのストレージに保存されたスキャン資料を有効活用することを目的として設計されたシステムです。 

- Library/Collection/Bookを単位としてスキャン資料を自由に整理できます。
- 書籍やページには永続性のある静的なURLが与えられ、Wikiなどから自由に参照できます。
- 各書籍には、BibTeXと互換性のある書誌情報の他、目次情報を保存できます。
- 各画像ページには、タグ・メモ・マーカーなどの注釈情報を保存できます。

## Introduction

### 問題

- 現在のオープンソース（OSS）書籍ビューアの多くは、基本的な表示・整理機能に特化しており、研究や探究に不可欠なメモ、マーカー、タグ付けといった注釈機能の統合が不十分である。
- 技術的側面では、シングルページアプリケーション（SPA）として構築されたシステムが主流であり、各ページに対するURIの永続性や参照可能性への配慮が欠けている。このため、特定の箇所へのリンクが将来的に損なわれるリスクを内包しており、知的生活の情報基盤としての信頼性に課題が残る。
- こうした開発状況の背景には、欧米諸国において書籍のデジタル化に対する法的な制限が厳しく、日本のような独自の自炊文化や、デジタル資料をOSSで積極的に利活用しようとする文化が醸成されにくいという社会的要因も推察される。
- したがってデジタル書籍資料を単なる消費財ではなく、永続的な知的資源として扱うための新たな情報システムの開発が必要とされている。


### 非目標
- 二人以上の人と共有して使用することはこのアプリの目標ではない。
- アプリを世界に一般公開することは想定していない。そもそも、スキャンした書籍を一般公開することは絶対にしてはいけないし、スキャンすること自体が違法な国もある。

## 要件
物理本との比較によるユーザー体験要件

- **安心感の継承**: 本棚の本は失われないという安心感を損ねない
- **アクセス性の向上**:
  - ブラウジング: 本をパラパラめくる体験をシミュレート
  - 目次: 情報構造の全体像を素早く把握
  - タグ・マーク: ページへのクイックアクセス
  - 書き込み: 記憶の想起トリガー
- **永続的リソースアクセス**: すべてのページにURIを割り当て
- **抽出可能性**: 内容を他のツールを関連付け可能に

## 機能
上記要件を実装するための具体的機能

### コアビューア機能
- **ページ表示**: スキャン画像を各ページにURIを付与して配信
- **閲覧**: ホイールズーム、ドラッグパン、キーボードナビゲーション

### ナビゲーション
- **セクション/目次**: 資料にセクションを登録し、セクション間の移動
- **ページジャンプ**: ページ番号指定での直接アクセス

### ページ注釈
- **マークダウンノート**: 要約・コメント・メモをMarkdown形式で作成
- **手描き注釈**: SVGペンで図解・強調などの描画
- **ステータスタグ**: ページごとの「既読/途中/未読」などのステータス管理

### Library / Collection / Book モデル

- LibraryとCollcetionは、本を整理するための単位であり、Libraryは複数のCollectionをグループ化し、Collectionは複数のBookをグループ化する。
- LibraryとCollectionの子要素は複数のグループに所属することができる。例えばあるBookは複数のCollectionに所属できるし、あるCollectionは複数のLibraryに所属できる。
- ただし、あるBookを直接Libraryに所属させることはできず、かならずCollectionを介してLibraryに所属させる必要がある。
- また、すべてのBookは必ず"All Books" Collectionに所属し、他に所属しているCollectionがなければ自動的に"Uncategorized Books" Collectionに追加される。つまり、すべてのBookは2つ以上のCollectionに必ず所属することになる。
- 同様に、すべてのCollectionは必ず"All Collections" Libraryに所属し、他に所属しているLibraryがなければ自動的に"Uncategorized Collections" Libraryに追加される。つまり、すべてのCollectionは2つ以上のLibraryに必ず所属することになる。
- "All Books"と"Uncategorized Books" Collectionは、必ず"Central Library"に所属する。
- "Central Library" は削除不可であり、All Collections / Uncategorized Collections も削除不可。"All Books" / "Uncategorized Books" Collection も同様に削除不可。


### その他
- **書籍ノート**: 書籍レベルのメモ（全体的な感想など）
- **進捗率表示**: 書籍全体のどの位置を読んでいるかを可視化


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
│   ├── main.go        # CLIエントリーポイント; サブコマンドの振り分け
│   └── server.go      # HTTPサーバー設定とルート登録
├── internal/
│   ├── config/
│   │   └── config.go  # 環境変数の読み込み; Config構造体
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
│   │   ├── types.go      # Book構造体とImageEntry構造体
│   │   ├── cbz.go        # CBZのオープン、folio.jsonの読み書き、画像一覧取得、ページ配信
│   │   ├── scan.go       # ライブラリの再帰走査; ワーカープールによるScanおよびScanMeta
│   │   └── thumbnail.go  # 書籍レベルおよびページレベルのJPEGサムネイル生成
│   └── store/
│       ├── store.go    # SQLiteのオープン、スキーマ初期化、マイグレーション適用
│       └── queries.go  # すべてのDB読み書き操作; ドメイン型定義
├── src/                        # TypeScriptおよびCSSソース
│   ├── main.ts                 # DOMContentLoaded初期化ディスパッチャ
│   ├── api.ts                  # すべてのRESTエンドポイント用の集中化されたfetchヘルパー
│   ├── types.ts                # 共有フロントエンドドメイン型 (ReadStatus, NotePayload など)
│   ├── viewer/
│   │   ├── navigation.ts  # キーボードナビゲーション、ページジャンプセレクタ
│   │   ├── display.ts     # ホイールズーム、ドラッグパン、ダブルクリックリセット
│   │   ├── editor.ts      # 編集ペインの開閉、メモ保存、スナップショット/復元
│   │   ├── toc.ts         # TOCペイン切り替え
│   │   └── drawing.ts     # SVG描画オーバーレイ、ペン/消しゴム、undo/redo、保存
│   ├── ui/
│   │   ├── search.ts        # 書籍グリッドのタイトルフィルタ
│   │   ├── rename.ts        # 編集モードでの書籍タイトルのインラインリネーム
│   │   ├── collections.ts   # サイドバードラッグドロップ、複数選択、作成/リネーム/削除
│   │   ├── page-status.ts   # ページごとの既読ステータスボタン
│   │   ├── book-note.ts     # 書籍レベルのメモ保存
│   │   └── components.ts    # 将来のtoast/modal UI要素用スタブ
│   ├── css/
│   │   ├── base.css      # デザイントークン (CSS変数)、リセット、サイトヘッダ
│   │   ├── pane.css      # 共通スライドインペイン構造 (TOCと編集ペイン)
│   │   ├── shelf.css     # ライブラリグリッド、書籍カード、検索バー、欠落書籍
│   │   ├── sidebar.css   # コレクションサイドバー、ドラッグオーバー状態、複数選択
│   │   ├── viewer.css    # ビューワーレイアウト、画像表示、メモ表示、ジャンプオーバーレイ
│   │   ├── toc.css       # TOCペイン内容スタイル
│   │   ├── editor.css    # 編集ペインフォームスタイル
│   │   ├── book.css      # 書籍ごとのページグリッドおよびページカードスタイル
│   │   ├── drawing.css   # SVGオーバーレイ、描画ペイン、ツールボタン
│   │   └── overview.css  # 概要ページのタブナビ、ステータス着色、書誌レイアウト
│   ├── style.css         # CSSエントリーポイント; css/* のすべてをimport
│   └── folio.svg         # アプリケーションアイコン元データ
├── templates/
│   ├── layout.html        # ベースHTMLシェル; title と content ブロック定義
│   ├── sidebar.html       # コレクションサイドバー部分テンプレート; home と collection から読み込み
│   ├── home.html          # 全書籍ライブラリページ
│   ├── collection.html    # 単一コレクションの書籍一覧
│   ├── overview.html      # 書籍ごとのページグリッド（ステータスボタン付き）
│   ├── bibliography.html # 書籍ごとのTOC、統計、書籍レベルメモ
│   └── viewer.html        # TOC・編集・描画ペイン付き単ページビューア
├── docs/
│   ├── design-01.md  # 初期設計 (廃止済み)
│   ├── design-02.md  # データ所有哲学 (廃止済み)
│   ├── design-03.md  # Phase 3スキーマ (廃止済み)
│   └── design-04.md  # 現在の設計リファレンス (このドキュメント)
├── static/           # ビルド出力 (favicon.ico を除き gitignore)
├── Makefile          # build, watch, docker, icon, clean ターゲット
├── go.mod / go.sum   # Goモジュール定義とチェックサム
├── shell.nix         # Nix開発シェル
├── .air.toml         # Airライブリロード設定
├── folio.env         # ローカル環境変数デフォルト
└── .gitignore
```