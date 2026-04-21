# Folio

<img src="src/icons/folio.svg" width="90" align="right" />

Folioはローカルサーバのストレージに保存されたスキャン資料を有効活用することを目的として設計されたシステムです。 

- Library/Collection/Bookを単位としてスキャン資料を自由に整理できます。
- 書籍やページには永続性のある静的なURLが与えられ、Wikiなどから自由に参照できます。
- 各書籍には、BibTeXと互換性のある書誌情報の他、目次情報を保存できます。
- 各画像ページには、タグ・メモ・マーカーなどの注釈情報を保存できます。



### 注意点
- Folioは探究心のある個人のために作られたセルフホスティングアプリであり一般公開やコラボレーションを想定していない。


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

