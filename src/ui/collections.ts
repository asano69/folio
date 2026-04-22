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

// ── Sidebar filter state ───────────────────────────────────────
// Both filters are applied together so they don't override each other.
let activeLibraryID = '';   // '' = Central Library (show all)
let collectionTextQuery = '';

export function initCollections(): void {
  setupCollectionFilter();
  setupLibrarySwitcher();
  setupMultiSelect();
  setupDragAndDrop();
  setupEditMode();
  setupRemoveFromCollection();
}

// ── Collection filter ──────────────────────────────────────────

function applyCollectionFilters(): void {
  document.querySelectorAll<HTMLElement>('.collection-drop-zone').forEach(item => {
    const title = (item.querySelector('.collection-title')?.textContent ?? '').toLowerCase();
    const libIds = (item.dataset.libraryIds ?? '').split(',').filter(Boolean);

    const matchesText = !collectionTextQuery || title.includes(collectionTextQuery);
    const matchesLib  = !activeLibraryID || libIds.includes(activeLibraryID);

    item.style.display = (matchesText && matchesLib) ? '' : 'none';
  });

  // All Books and Uncategorized belong to Central Library only.
  document.querySelectorAll<HTMLElement>('[data-central-only]').forEach(item => {
    item.style.display = activeLibraryID ? 'none' : '';
  });
}

function setupCollectionFilter(): void {
  const input = document.getElementById('collection-search') as HTMLInputElement | null;
  if (!input) return;

  input.addEventListener('input', () => {
    collectionTextQuery = input.value.trim().toLowerCase();
    applyCollectionFilters();
  });
  input.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Escape') { input.value = ''; collectionTextQuery = ''; applyCollectionFilters(); }
  });
}

function setupLibrarySwitcher(): void {
  const select = document.getElementById('library-select') as HTMLSelectElement | null;
  if (!select) return;

  const centralID = select.querySelector<HTMLOptionElement>('[data-is-central]')?.value ?? '';

  select.addEventListener('change', () => {
    activeLibraryID = select.value === centralID ? '' : select.value;
    applyCollectionFilters();
  });
}

// ── Multi-select ───────────────────────────────────────────────

function setupMultiSelect(): void {
  const grid = document.querySelector<HTMLElement>('.books-grid:not(.missing-grid)');
  if (!grid) return;

  selectionBadge = document.createElement('div');
  selectionBadge.className = 'selection-badge';
  selectionBadge.hidden = true;
  document.body.appendChild(selectionBadge);

  // Ctrl/Cmd+click toggles individual card selection without navigating.
  document.querySelectorAll<HTMLElement>('.book-card[data-book-id]').forEach(card => {
    card.addEventListener('click', (e: MouseEvent) => {
      if (suppressNextClick) {
        e.preventDefault();
        suppressNextClick = false;
        return;
      }
      if (!!document.querySelector('.shelf--edit')) return;
      if (!(e.ctrlKey || e.metaKey)) return;
      e.preventDefault();
      toggleCardSelection(card);
    });
  });

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

// setupRubberBand enables click-drag on the grid background to select multiple cards.
function setupRubberBand(grid: HTMLElement): void {
  const booksMain = grid.closest<HTMLElement>('.books-main') ?? document.documentElement;
  let band: HTMLElement | null = null;
  let startX = 0;
  let startY = 0;
  let active = false;

  booksMain.addEventListener('mousedown', (e: MouseEvent) => {
    if (e.button !== 0) return;
    if (e.ctrlKey || e.metaKey) return;
    if (!!document.querySelector('.shelf--edit')) return;

    const target = e.target as HTMLElement;
    if (target.closest('.book-card') || target.closest('button') || target.closest('input')) return;

    startX = e.clientX;
    startY = e.clientY;
    active = false;

    const onMove = (e: MouseEvent): void => {
      const dx = e.clientX - startX;
      const dy = e.clientY - startY;
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
        if (active) {
          suppressNextClick = true;
          setTimeout(() => { suppressNextClick = false; }, 100);
        }
      }
      active = false;
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };

    e.preventDefault();
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
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
      const collectionId = zone.dataset.collectionId;
      if (bookId && collectionId) handleDrop(zone, collectionId, bookId);
    });
  });
}

// handleDrop adds the dragged book — or all selected books — to the target collection.
async function handleDrop(zone: HTMLElement, collectionId: string, bookId: string): Promise<void> {
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
      const collectionId = zone.dataset.collectionId;
      if (collectionId) startRenameCollection(collectionId, titleEl);
    });
  });

  document.querySelectorAll<HTMLButtonElement>('.collection-delete-btn').forEach(btn => {
    btn.addEventListener('click', async (e: Event) => {
      e.preventDefault();
      e.stopPropagation();
      const item = btn.closest<HTMLElement>('.collection-drop-zone');
      if (!item) return;
      const collectionId = item.dataset.collectionId;
      if (!collectionId) return;
      try {
        await deleteCollection(collectionId);
        if (window.location.pathname === `/collections/${collectionId}`) {
          window.location.href = '/collections/all';
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
    const name = input.value.trim();
    if (name) {
      try {
        await createCollection(name);
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

async function startRenameCollection(collectionId: string, titleEl: HTMLElement): Promise<void> {
  const currentName = titleEl.textContent ?? '';

  const input = document.createElement('input');
  input.type = 'text';
  input.value = currentName;
  input.className = 'collection-rename-input';
  titleEl.replaceWith(input);
  input.focus();
  input.select();

  let cancelled = false;
  let finishing = false;

  const finish = async (): Promise<void> => {
    if (finishing) return;
    finishing = true;
    const newName = input.value.trim();
    if (!cancelled && newName && newName !== currentName) {
      try {
        await renameCollection(collectionId, newName);
        titleEl.textContent = newName;
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
      const { bookId, collectionId } = btn.dataset;
      if (!bookId || !collectionId) return;
      try {
        await removeBookFromCollection(collectionId, bookId);
        btn.closest<HTMLElement>('.book-card')?.remove();
      } catch (err) {
        console.error(err);
      }
    });
  });
}
