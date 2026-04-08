# CLAUDE.md

## OpenBook Project Overview

サーバのファイルシステムに保存されたスキャン画像をベースにしたページ管理・注釈システムを実装する。
各ページに対してメタデータ・注釈・ナビゲーション構造を追加できるようにする。
スキャン画像自体は変更せず、メタデータのみを管理する軽量なデジタル書籍ビューア兼ノートシステムとする。

## Tech Stack

* Backend: Go
* Template Engine: html/template (go-template)
* Frontend: Server Side Rendering
* TypeScript: 必要最小限（描画・UI操作のみ）
* Storage: ローカルファイルベース
* Database: Sqlite3

## 開発戦略
* go標準モジュールのflagで、openbookコマンドを作り、openbook serverでサーバが起動するようにする。
* サーバの設定は環境変数を使い宣言的に行うことができるようにする。
