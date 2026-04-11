// src/keyboard/init.ts

import { keyboardManager, KeyBindingContext } from './keybindings';

// ── Initialize keyboard bindings ───────────────────────────────

export function initKeyboard(): void {
  // Global listener for all keyboard events.
  document.addEventListener('keydown', (e: KeyboardEvent) => {
    const context = buildContext();
    keyboardManager.handleKeydown(e, context);
  });
}

// buildContext constructs the current KeyBindingContext from DOM state.
function buildContext(): KeyBindingContext {
  return {
    isDrawPaneOpen: document.getElementById('draw-pane')?.classList.contains('open') ?? false,
    isEditPaneOpen: document.getElementById('edit-pane')?.classList.contains('open') ?? false,
    isTOCPaneOpen: document.getElementById('toc-pane')?.classList.contains('open') ?? false,
    isViewerPage: !!document.querySelector('.viewer-layout'),
    focusedElement: document.activeElement as HTMLElement | null,
  };
}

// ── Register all keybindings ───────────────────────────────────

import {
  onlyOnViewerPage,
  notWhenEditingText,
  whenDrawPaneOpen,
  whenEditPaneOpen,
  whenTOCPaneOpen,
  whenNoPaneOpen,
} from './keybindings';

// Home key
keyboardManager.register({
  id: 'nav-home',
  description: 'Return to library',
  keys: ['ctrl+h'],
  handler: () => {
    window.location.href = '/';
  },
  shouldFire: (ctx) => ctx.isViewerPage && notWhenEditingText(ctx),
});

// Jump to page
keyboardManager.register({
  id: 'viewer-jump',
  description: 'Jump to page',
  keys: ['ctrl+j'],
  handler: () => {
    const controls = document.querySelector('.viewer-controls') as HTMLElement | null;
    if (controls) {
      const opening = !controls.classList.contains('visible');
      controls.classList.toggle('visible');
      if (opening) {
        const sel = controls.querySelector('select') as HTMLSelectElement | null;
        sel?.focus();
      }
    }
  },
  shouldFire: (ctx) => ctx.isViewerPage && notWhenEditingText(ctx),
});

// Previous page
keyboardManager.register({
  id: 'viewer-prev-arrow',
  description: 'Previous page',
  keys: ['arrowleft'],
  handler: () => {
    const prevBtn = document.querySelector('.prev-btn:not(.disabled)') as HTMLAnchorElement;
    if (prevBtn) window.location.href = prevBtn.href;
  },
  shouldFire: (ctx) => ctx.isViewerPage && whenNoPaneOpen(ctx) && notWhenEditingText(ctx),
});

keyboardManager.register({
  id: 'viewer-prev-h',
  description: 'Previous page (vim-style)',
  keys: ['h'],
  handler: () => {
    const prevBtn = document.querySelector('.prev-btn:not(.disabled)') as HTMLAnchorElement;
    if (prevBtn) window.location.href = prevBtn.href;
  },
  shouldFire: (ctx) => ctx.isViewerPage && whenNoPaneOpen(ctx) && notWhenEditingText(ctx),
});

// Next page
keyboardManager.register({
  id: 'viewer-next-arrow',
  description: 'Next page',
  keys: ['arrowright'],
  handler: () => {
    const nextBtn = document.querySelector('.next-btn:not(.disabled)') as HTMLAnchorElement;
    if (nextBtn) window.location.href = nextBtn.href;
  },
  shouldFire: (ctx) => ctx.isViewerPage && whenNoPaneOpen(ctx) && notWhenEditingText(ctx),
});

keyboardManager.register({
  id: 'viewer-next-l',
  description: 'Next page (vim-style)',
  keys: ['l'],
  handler: () => {
    const nextBtn = document.querySelector('.next-btn:not(.disabled)') as HTMLAnchorElement;
    if (nextBtn) window.location.href = nextBtn.href;
  },
  shouldFire: (ctx) => ctx.isViewerPage && whenNoPaneOpen(ctx) && notWhenEditingText(ctx),
});

// Close any open pane with Escape
keyboardManager.register({
  id: 'pane-close',
  description: 'Close pane',
  keys: ['escape'],
  handler: () => {
    document.querySelector('.draw-pane.open')?.classList.remove('open');
    document.querySelector('.edit-pane.open')?.classList.remove('open');
    document.querySelector('.toc-pane.open')?.classList.remove('open');
    document.querySelectorAll('.pane-backdrop.visible').forEach(el => {
      el.classList.remove('visible');
    });
  },
  shouldFire: (ctx) => (ctx.isDrawPaneOpen || ctx.isEditPaneOpen || ctx.isTOCPaneOpen),
});

// Undo in draw pane
keyboardManager.register({
  id: 'draw-undo',
  description: 'Undo drawing',
  keys: ['ctrl+z'],
  handler: (e) => {
    // The drawing module will handle the actual undo.
    // This binding simply ensures Ctrl+Z is handled by the drawing module
    // when the draw pane is open, not by the browser.
    document.dispatchEvent(new CustomEvent('folio:draw-undo'));
  },
  shouldFire: (ctx) => ctx.isDrawPaneOpen && notWhenEditingText(ctx),
});

// Redo in draw pane
keyboardManager.register({
  id: 'draw-redo',
  description: 'Redo drawing',
  keys: ['ctrl+y', 'ctrl+shift+z'],
  handler: (e) => {
    document.dispatchEvent(new CustomEvent('folio:draw-redo'));
  },
  shouldFire: (ctx) => ctx.isDrawPaneOpen && notWhenEditingText(ctx),
});

// TOC toggle
keyboardManager.register({
  id: 'viewer-toc-toggle',
  description: 'Toggle table of contents',
  keys: ['ctrl+t'],
  handler: () => {
    const tocToggle = document.getElementById('toc-toggle') as HTMLButtonElement;
    if (tocToggle) tocToggle.click();
  },
  shouldFire: (ctx) => ctx.isViewerPage && notWhenEditingText(ctx),
});
