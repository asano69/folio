# CSS


## デザイントークン体系

### 色（18色）

```css
/* Brand colors */
--color-primary:       #99AD7A;      /* メイン色 */
--color-primary-dark:  #546B41;      /* ダークバリエーション */
--color-primary-light: #C3CC9B;      /* ライトバリエーション */

/* Neutrals & Grays */
--color-page-bg:       #F9F8F6;      /* ページ背景 */
--color-header-bg:     #EFE9E3;      /* ヘッダー背景 */
--color-surface-bg:    #fff;         /* サーフェス */
--color-surface-subtle:#F1F3E0;      /* サブトル背景 */
--color-hover-bg:      #f5f5f5;      /* ホバー状態 */

/* Text colors (grayscale) */
--color-text-primary:    #333;
--color-text-secondary:  #444;
--color-text-tertiary:   #555;
--color-text-light:      #666;
--color-text-muted:      #888;
--color-text-faint:      #aaa;
--color-text-disabled:   #bbb;

/* Borders & dividers */
--color-border:        #e0e0e0;
--color-border-light:  #e8e8e8;
--color-border-medium: #ccc;

/* Semantic */
--color-danger:        #9B0F06;
```

### タイポグラフィ

**フォントサイズ（7段階）**
```css
--font-size-xs:   0.75rem;   /* 12px */
--font-size-sm:   0.85rem;   /* 13.6px */
--font-size-md:   0.95rem;   /* 15.2px */
--font-size-base: 1rem;      /* 16px */
--font-size-lg:   1.1rem;    /* 17.6px */
--font-size-xl:   1.5rem;    /* 24px */
--font-size-2xl:  1.8rem;    /* 28.8px */
```

**行高**
```css
--line-height-tight:    1;
--line-height-base:     1.6;
--line-height-relaxed:  1.8;
```

### 間隔（9段階）

```css
--spacing-xs:   0.15rem;  /* 2.4px */
--spacing-sm:   0.25rem;  /* 4px */
--spacing-md:   0.5rem;   /* 8px */
--spacing-lg:   0.75rem;  /* 12px */
--spacing-xl:   1rem;     /* 16px */
--spacing-2xl:  1.25rem;  /* 20px */
--spacing-3xl:  1.5rem;   /* 24px */
--spacing-4xl:  2rem;     /* 32px */
--spacing-5xl:  2.5rem;   /* 40px */
```

### Shape

```css
--radius-sm:  3px;
--radius:     4px;
--radius-lg:  8px;
```

### Shadows

```css
--shadow-sm: 0 2px 4px rgba(0, 0, 0, 0.05);
--shadow-md: 0 2px 8px rgba(0, 0, 0, 0.1);
--shadow-lg: 0 4px 12px rgba(0, 0, 0, 0.15);
```

### Motion

```css
--transition-fast: 0.15s;
--transition-pane: 0.25s ease;
```

### Z-index

```css
--z-backdrop: 99;
--z-pane:     100;
--z-overlay:  101;
```


## ファイル構成

```
src/css/
├── base.css       ← 全デザイントークン定義（ここから開始）
├── pane.css       ← ペイン系共通（TOC、Edit、Draw）
├── sidebar.css    ← コレクションサイドバー
├── shelf.css      ← 書籍グリッド・ライブラリレイアウト
├── viewer.css     ← ページビューア
├── toc.css        ← TOCペイン専用
├── editor.css     ← 編集ペイン専用
├── drawing.css    ← 描画ペイン専用
├── book.css       ← ページグリッド表示
└── overview.css   ← 概要ページ・書誌情報

style.css         ← エントリーポイント（import順序管理）
```



## 🔄 Import順序（style.css）

```css
/* 1. Foundation: Design tokens (必須・最初) */
@import "./css/base.css";

/* 2. Base components: Panes and layouts */
@import "./css/pane.css";

/* 3. Page layouts */
@import "./css/sidebar.css";
@import "./css/shelf.css";
@import "./css/viewer.css";

/* 4. Viewer sub-panes */
@import "./css/toc.css";
@import "./css/editor.css";
@import "./css/drawing.css";

/* 5. Page-specific components */
@import "./css/book.css";
@import "./css/overview.css";
```

**理由**：
- `base.css` が最初：すべてのトークンが先に定義される
- `pane.css` が次：複数のペインが依存するため
- レイアウト系 → コンポーネント系 → ページ固有順

