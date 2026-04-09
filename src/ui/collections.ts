// Collections sidebar: drag-and-drop, create, rename, delete, and filter.
import {
  addBookToCollection,
  removeBookFromCollection,
  createCollection,
  renameCollection,
  deleteCollection,
} from '../api';

export function initCollections(): void {
  setupCollectionFilter();
  setupDragAndDrop();
  setupCreateCollection();
  setupCollectionActions();
  setupRemoveFromCollection();
}

// ── Collection filter ─────────────────────────────────────────

// Filters the named collection entries as the user types.
// "All Books" is always visible because it lacks the .collection-drop-zone class.
function setupCollectionFilter(): void {
  const input = document.getElementById('collection-search') as HTMLInputElement | null;
  if (!input) return;

  const applyFilter = (): void => {
    const query = input.value.trim().toLowerCase();
    document.querySelectorAll<HTMLElement>('.collection-drop-zone').forEach(item => {
      const title = item.querySelector<HTMLElement>('.collection-title')?.textContent ?? '';
      // Use style.display directly to avoid the CSS display:flex override on [hidden].
      item.style.display = (!query || title.toLowerCase().includes(query)) ? '' : 'none';
    });
  };

  input.addEventListener('input', applyFilter);

  // Escape clears the filter.
  input.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      input.value = '';
      applyFilter();
    }
  });
}

// ── Drag and drop ────────────────────────────────────────────

function setupDragAndDrop(): void {
  // Book cards are drag sources.
  document.querySelectorAll<HTMLElement>('.book-card[data-book-id]').forEach(card => {
    card.addEventListener('dragstart', (e: DragEvent) => {
      e.dataTransfer!.setData('text/plain', card.dataset.bookId!);
      e.dataTransfer!.effectAllowed = 'copy';
      card.classList.add('dragging');
    });
    card.addEventListener('dragend', () => card.classList.remove('dragging'));
  });

  // Collection items are drop targets.
  document.querySelectorAll<HTMLElement>('.collection-drop-zone').forEach(zone => {
    zone.addEventListener('dragover', (e: DragEvent) => {
      e.preventDefault();
      e.dataTransfer!.dropEffect = 'copy';
      zone.classList.add('drag-over');
    });
    zone.addEventListener('dragleave', () => zone.classList.remove('drag-over'));
    zone.addEventListener('drop', (e: DragEvent) => {
      e.preventDefault();
      zone.classList.remove('drag-over');
      const bookId = e.dataTransfer!.getData('text/plain');
      const collectionId = zone.dataset.collectionId!;
      if (bookId && collectionId) {
        handleDrop(zone, collectionId, bookId);
      }
    });
  });
}

async function handleDrop(
  zone: HTMLElement,
  collectionId: string,
  bookId: string,
): Promise<void> {
  try {
    const { added } = await addBookToCollection(collectionId, bookId);

    // Only increment the displayed count when the book was not already a member.
    if (added) {
      const countEl = zone.querySelector<HTMLElement>('.collection-count');
      if (countEl) {
        const n = parseInt(countEl.textContent?.match(/\d+/)?.[0] ?? '0', 10);
        countEl.textContent = `(${n + 1})`;
      }
    }

    zone.classList.add('drop-success');
    setTimeout(() => zone.classList.remove('drop-success'), 700);
  } catch (err) {
    console.error(err);
  }
}

// ── Create collection ─────────────────────────────────────────

function setupCreateCollection(): void {
  const btn = document.getElementById('collection-new-btn');
  if (!btn) return;

  btn.addEventListener('click', () => {
    const input = document.createElement('input');
    input.type = 'text';
    input.className = 'collection-new-input';
    input.placeholder = 'Collection name';

    btn.replaceWith(input);
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
      input.replaceWith(btn);
    };

    input.addEventListener('blur', finish);
    input.addEventListener('keydown', (e: KeyboardEvent) => {
      if (e.key === 'Enter') { e.preventDefault(); input.blur(); }
      if (e.key === 'Escape') { input.value = ''; input.blur(); }
    });
  });
}

// ── Rename and delete ─────────────────────────────────────────

function setupCollectionActions(): void {
  document.querySelectorAll<HTMLButtonElement>('.collection-rename-btn').forEach(btn => {
    btn.addEventListener('click', (e: Event) => {
      e.preventDefault();
      e.stopPropagation();
      const item = btn.closest<HTMLElement>('.collection-drop-zone');
      const titleEl = item?.querySelector<HTMLElement>('.collection-title');
      if (!item || !titleEl) return;
      startRenameCollection(item.dataset.collectionId!, titleEl);
    });
  });

  document.querySelectorAll<HTMLButtonElement>('.collection-delete-btn').forEach(btn => {
    btn.addEventListener('click', async (e: Event) => {
      e.preventDefault();
      e.stopPropagation();
      const item = btn.closest<HTMLElement>('.collection-drop-zone');
      if (!item) return;
      const collectionId = item.dataset.collectionId!;

      try {
        await deleteCollection(collectionId);

        // If the deleted collection is currently active, return to All Books.
        const params = new URLSearchParams(window.location.search);
        if (params.get('collection') === collectionId) {
          window.location.href = '/';
        } else {
          item.remove();
        }
      } catch (err) {
        console.error(err);
      }
    });
  });
}

async function startRenameCollection(
  collectionId: string,
  titleEl: HTMLElement,
): Promise<void> {
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

// ── Remove book from collection ───────────────────────────────

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
