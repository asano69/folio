// Collections sidebar: drag-and-drop, create, rename, delete, filter, and multi-select.
import {
  addBookToCollection,
  removeBookFromCollection,
  createCollection,
  renameCollection,
  deleteCollection,
} from '../api';

// ── Multi-select state ─────────────────────────────────────────
// Shared across drag-drop and rubber-band handlers.
const selectedBookIds = new Set<string>();
let selectionBadge: HTMLElement | null = null;
// Suppresses the click event that fires after a rubber-band drag ends.
let suppressNextClick = false;

export function initCollections(): void {
  setupCollectionFilter();
  setupMultiSelect();
  setupDragAndDrop();
  setupEditMode();
  setupRemoveFromCollection();
}

// ── Multi-select ───────────────────────────────────────────────

function setupMultiSelect(): void {
  const grid = document.querySelector<HTMLElement>('.books-grid:not(.missing-grid)');
  if (!grid) return;

  // Badge showing the count of currently selected books.
  selectionBadge = document.createElement('div');
  selectionBadge.className = 'selection-badge';
  selectionBadge.hidden = true;
  document.body.appendChild(selectionBadge);

  // Ctrl/Cmd+click toggles individual card selection without navigating.
  document.querySelectorAll<HTMLElement>('.book-card[data-book-id]').forEach(card => {
    card.addEventListener('click', (e: MouseEvent) => {
      if (suppressNextClick) {
        // Discard the click that fires immediately after a rubber-band drag ends.
        e.preventDefault();
        suppressNextClick = false;
        return;
      }
      if (!!document.querySelector('.shelf--edit')) return; // rename mode owns clicks
      if (!(e.ctrlKey || e.metaKey)) return;
      e.preventDefault();
      toggleCardSelection(card);
    });
  });

  // Escape clears the selection.
  document.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Escape') clearSelection();
  });

  setupRubberBand(grid);
}

function toggleCardSelection(card: HTMLElement): void {
  const bookId = card.dataset.bookId!;
  if (selectedBookIds.has(bookId)) {
    selectedBookIds.delete(bookId);
    card.classList.remove('book-card--selected');
  } else {
    selectedBookIds.add(bookId);
    card.classList.add('book-card--selected');
  }
  updateSelectionBadge();
}

function clearSelection(): void {
  selectedBookIds.clear();
  document.querySelectorAll<HTMLElement>('.book-card--selected').forEach(c =>
    c.classList.remove('book-card--selected')
  );
  updateSelectionBadge();
}

function updateSelectionBadge(): void {
  if (!selectionBadge) return;
  const n = selectedBookIds.size;
  selectionBadge.hidden = n === 0;
  if (n > 0) selectionBadge.textContent = `${n} book${n === 1 ? '' : 's'} selected`;
}

// setupRubberBand enables click-drag on the grid background to select multiple
// cards by drawing a selection rectangle.
function setupRubberBand(grid: HTMLElement): void {
  const booksMain = grid.closest<HTMLElement>('.books-main') ?? document.documentElement;
  let band: HTMLElement | null = null;
  let startX = 0;
  let startY = 0;
  let active = false;

  booksMain.addEventListener('mousedown', (e: MouseEvent) => {
    if (e.button !== 0) return;
    if (e.ctrlKey || e.metaKey) return; // Ctrl+click is handled per-card above
    if (!!document.querySelector('.shelf--edit')) return;

    const target = e.target as HTMLElement;
    // Only start rubber-band on the background — not on interactive elements.
    if (target.closest('.book-card') || target.closest('button') || target.closest('input')) return;

    startX = e.clientX;
    startY = e.clientY;
    active = false;

    const onMove = (e: MouseEvent): void => {
      const dx = e.clientX - startX;
      const dy = e.clientY - startY;
      // Wait for a small movement threshold before activating.
      if (!active && Math.hypot(dx, dy) < 6) return;

      if (!active) {
        active = true;
        clearSelection();
        band = document.createElement('div');
        band.className = 'rubber-band';
        document.body.appendChild(band);
      }

      const x = Math.min(e.clientX, startX);
      const y = Math.min(e.clientY, startY);
      const w = Math.abs(dx);
      const h = Math.abs(dy);
      band!.style.cssText = `left:${x}px;top:${y}px;width:${w}px;height:${h}px`;

      // Select cards whose bounding boxes intersect the rubber-band rectangle.
      const sel = { left: x, top: y, right: x + w, bottom: y + h };
      document.querySelectorAll<HTMLElement>(
        '.books-grid:not(.missing-grid) .book-card[data-book-id]'
      ).forEach(card => {
        const r = card.getBoundingClientRect();
        const hit =
          r.right > sel.left && r.left < sel.right &&
          r.bottom > sel.top  && r.top  < sel.bottom;
        card.classList.toggle('book-card--selected', hit);
        if (hit) selectedBookIds.add(card.dataset.bookId!);
        else     selectedBookIds.delete(card.dataset.bookId!);
      });
      updateSelectionBadge();
    };

    const onUp = (): void => {
      if (band) {
        band.remove();
        band = null;
        // Suppress the click event that the browser fires after mouseup when
        // the pointer happens to land on a card.
        if (active) {
          suppressNextClick = true;
          setTimeout(() => { suppressNextClick = false; }, 100);
        }
      }
      active = false;
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };

    // Prevent the browser from starting a text-selection drag.
    e.preventDefault();
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  });
}

// ── Collection filter ──────────────────────────────────────────

function setupCollectionFilter(): void {
  const input = document.getElementById('collection-search') as HTMLInputElement | null;
  if (!input) return;

  const applyFilter = (): void => {
    const query = input.value.trim().toLowerCase();
    document.querySelectorAll<HTMLElement>('.collection-drop-zone').forEach(item => {
      const title = item.querySelector<HTMLElement>('.collection-title')?.textContent ?? '';
      item.style.display = (!query || title.toLowerCase().includes(query)) ? '' : 'none';
    });
  };

  input.addEventListener('input', applyFilter);
  input.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Escape') { input.value = ''; applyFilter(); }
  });
}

// ── Drag and drop ──────────────────────────────────────────────

function setupDragAndDrop(): void {
  document.querySelectorAll<HTMLElement>('.book-card[data-book-id]').forEach(card => {
    card.addEventListener('dragstart', (e: DragEvent) => {
      e.dataTransfer!.setData('text/plain', card.dataset.bookId!);
      e.dataTransfer!.effectAllowed = 'copy';
      card.classList.add('dragging');
    });
    card.addEventListener('dragend', () => {
      card.classList.remove('dragging');
      document.querySelectorAll<HTMLElement>('.collection-drop-zone.drag-over').forEach(z => {
        z.classList.remove('drag-over');
      });
    });
  });

  document.querySelectorAll<HTMLElement>('.collection-drop-zone').forEach(zone => {
    zone.addEventListener('dragover', (e: DragEvent) => {
      e.preventDefault();
      e.dataTransfer!.dropEffect = 'copy';
      zone.classList.add('drag-over');
    });

    zone.addEventListener('dragleave', (e: DragEvent) => {
      if (!zone.contains(e.relatedTarget as Node)) zone.classList.remove('drag-over');
    });

    zone.addEventListener('drop', (e: DragEvent) => {
      e.preventDefault();
      zone.classList.remove('drag-over');
      const bookId = e.dataTransfer!.getData('text/plain');
      // Parse to number immediately on extraction from the DOM so the type
      // matches the api.ts signature and Collection.id throughout.
      const collectionId = parseInt(zone.dataset.collectionId!, 10);
      if (bookId && !isNaN(collectionId)) handleDrop(zone, collectionId, bookId);
    });
  });
}

// handleDrop adds the dragged book — or all selected books when the dragged
// card belongs to the current selection — to the target collection.
async function handleDrop(zone: HTMLElement, collectionId: number, bookId: string): Promise<void> {
  const idsToAdd =
    selectedBookIds.has(bookId) && selectedBookIds.size > 0
      ? [...selectedBookIds]
      : [bookId];

  let addedCount = 0;
  for (const id of idsToAdd) {
    try {
      const { added } = await addBookToCollection(collectionId, id);
      if (added) addedCount++;
    } catch (err) {
      console.error(err);
    }
  }

  if (addedCount > 0) {
    const countEl = zone.querySelector<HTMLElement>('.collection-count');
    if (countEl) {
      const n = parseInt(countEl.textContent?.match(/\d+/)?.[0] ?? '0', 10);
      countEl.textContent = `(${n + addedCount})`;
    }
  }

  zone.classList.add('drop-success');
  setTimeout(() => zone.classList.remove('drop-success'), 700);
}

// ── Edit mode ──────────────────────────────────────────────────

function setupEditMode(): void {
  const editBtn = document.getElementById('collection-edit-btn') as HTMLButtonElement | null;
  const addItem = document.getElementById('collection-add-item') as HTMLElement | null;

  if (!editBtn) return;

  let editMode = false;

  const setEditMode = (active: boolean): void => {
    editMode = active;
    editBtn.classList.toggle('active', active);
    document.querySelectorAll<HTMLElement>('.collection-delete-btn').forEach(btn => {
      btn.hidden = !active;
    });
    if (addItem) addItem.hidden = !active;
  };

  editBtn.addEventListener('click', () => setEditMode(!editMode));

  document.querySelectorAll<HTMLElement>('.collection-drop-zone').forEach(zone => {
    const link = zone.querySelector<HTMLElement>('.collection-link');
    link?.addEventListener('click', (e: Event) => {
      if (!editMode) return;
      e.preventDefault();
      const titleEl = zone.querySelector<HTMLElement>('.collection-title');
      if (!titleEl) return;
      // Parse to number immediately on extraction from the DOM.
      const collectionId = parseInt(zone.dataset.collectionId!, 10);
      if (!isNaN(collectionId)) startRenameCollection(collectionId, titleEl);
    });
  });

  document.querySelectorAll<HTMLButtonElement>('.collection-delete-btn').forEach(btn => {
    btn.addEventListener('click', async (e: Event) => {
      e.preventDefault();
      e.stopPropagation();
      const item = btn.closest<HTMLElement>('.collection-drop-zone');
      if (!item) return;
      // Parse to number immediately on extraction from the DOM.
      const collectionId = parseInt(item.dataset.collectionId!, 10);
      if (isNaN(collectionId)) return;
      try {
        await deleteCollection(collectionId);
        // If we are currently viewing this collection, redirect to home.
        if (window.location.pathname === `/collections/${collectionId}`) {
          window.location.href = '/';
        } else {
          item.remove();
        }
      } catch (err) {
        console.error(err);
      }
    });
  });

  addItem?.addEventListener('click', () => startCreateCollection(addItem));
}

async function startCreateCollection(addItem: HTMLElement): Promise<void> {
  const label = addItem.querySelector<HTMLElement>('.collection-add-label');
  if (!label) return;

  const input = document.createElement('input');
  input.type = 'text';
  input.className = 'collection-new-input';
  input.placeholder = 'Collection name';
  label.replaceWith(input);
  input.focus();

  let finishing = false;

  const finish = async (): Promise<void> => {
    if (finishing) return;
    finishing = true;
    const title = input.value.trim();
    if (title) {
      try {
        await createCollection(title);
        window.location.reload();
        return;
      } catch (err) {
        console.error(err);
      }
    }
    input.replaceWith(label);
  };

  input.addEventListener('blur', finish);
  input.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Enter') { e.preventDefault(); input.blur(); }
    if (e.key === 'Escape') { input.value = ''; input.blur(); }
  });
}

async function startRenameCollection(collectionId: number, titleEl: HTMLElement): Promise<void> {
  const currentTitle = titleEl.textContent ?? '';

  const input = document.createElement('input');
  input.type = 'text';
  input.value = currentTitle;
  input.className = 'collection-rename-input';
  titleEl.replaceWith(input);
  input.focus();
  input.select();

  let cancelled = false;
  let finishing = false;

  const finish = async (): Promise<void> => {
    if (finishing) return;
    finishing = true;
    const newTitle = input.value.trim();
    if (!cancelled && newTitle && newTitle !== currentTitle) {
      try {
        await renameCollection(collectionId, newTitle);
        titleEl.textContent = newTitle;
      } catch (err) {
        console.error(err);
      }
    }
    input.replaceWith(titleEl);
  };

  input.addEventListener('blur', finish);
  input.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Enter') { e.preventDefault(); input.blur(); }
    if (e.key === 'Escape') { cancelled = true; input.blur(); }
  });
}

// ── Remove book from collection ────────────────────────────────

function setupRemoveFromCollection(): void {
  document.querySelectorAll<HTMLButtonElement>('.collection-remove-btn').forEach(btn => {
    btn.addEventListener('click', async (e: Event) => {
      e.preventDefault();
      e.stopPropagation();
      const { bookId } = btn.dataset;
      // Parse to number immediately on extraction from the DOM.
      const collectionId = parseInt(btn.dataset.collectionId!, 10);
      if (!bookId || isNaN(collectionId)) return;
      try {
        await removeBookFromCollection(collectionId, bookId);
        btn.closest<HTMLElement>('.book-card')?.remove();
      } catch (err) {
        console.error(err);
      }
    });
  });
}
