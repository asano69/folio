# Library / Collection / Book モデル

- LibraryとCollcetionは、本を整理するための単位であり、Libraryは複数のCollectionをグループ化し、Collectionは複数のBookをグループ化する。
- LibraryとCollectionの子要素は複数のグループに所属することができる。例えばあるBookは複数のCollectionに所属できるし、あるCollectionは複数のLibraryに所属できる。
- ただし、あるBookを直接Libraryに所属させることはできず、かならずCollectionを介してLibraryに所属させる必要がある。
- また、すべてのBookは必ず"All Books" Collectionに所属し、他に所属しているCollectionがなければ自動的に"Uncategorized Books" Collectionに追加される。つまり、すべてのBookは2つ以上のCollectionに必ず所属することになる。
- 同様に、すべてのCollectionは必ず"All Collections" Libraryに所属し、他に所属しているLibraryがなければ自動的に"Uncategorized Collections" Libraryに追加される。つまり、すべてのCollectionは2つ以上のLibraryに必ず所属することになる。
- "All Books"と"Uncategorized Books" Collectionは、必ず"Central Library"に所属する。
- "Central Library" は削除不可であり、All Collections / Uncategorized Collections も削除不可。"All Books" / "Uncategorized Books" Collection も同様に削除不可。