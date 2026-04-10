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
  setupEditMode();
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
    card.addEventListener('dragend', () => {
      card.classList.remove('dragging');
      // Clean up any stale drag-over state (e.g. when the drag is cancelled
      // outside the browser window and dragleave does not fire).
      document.querySelectorAll<HTMLElement>('.collection-drop-zone.drag-over').forEach(z => {
        z.classList.remove('drag-over');
      });
    });
  });

  // Collection items are drop targets.
  document.querySelectorAll<HTMLElement>('.collection-drop-zone').forEach(zone => {
    zone.addEventListener('dragover', (e: DragEvent) => {
      e.preventDefault();
      e.dataTransfer!.dropEffect = 'copy';
      zone.classList.add('drag-over');
    });

    // Chrome fires dragleave when the pointer moves over a child element.
    // Only remove the class when the pointer has truly left the zone.
    zone.addEventListener('dragleave', (e: DragEvent) => {
      if (!zone.contains(e.relatedTarget as Node)) {
        zone.classList.remove('drag-over');
      }
    });

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

// ── Edit mode ─────────────────────────────────────────────────
//
// The edit button above the search box toggles edit mode.
// In edit mode:
//   - A delete (✕) button appears on the left of each collection item.
//   - Clicking the collection title starts an inline rename.
//   - An "Add Collection" item appears at the bottom of the list.

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

  // Clicking the collection link renames in edit mode; navigates normally otherwise.
  document.querySelectorAll<HTMLElement>('.collection-drop-zone').forEach(zone => {
    const link = zone.querySelector<HTMLElement>('.collection-link');
    link?.addEventListener('click', (e: Event) => {
      if (!editMode) return;
      e.preventDefault();
      const titleEl = zone.querySelector<HTMLElement>('.collection-title');
      if (!titleEl) return;
      startRenameCollection(zone.dataset.collectionId!, titleEl);
    });
  });

  // Delete buttons.
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

  // Add new collection inline.
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
