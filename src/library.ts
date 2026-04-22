// Library admin page interactions: CRUD for libraries and drag-and-drop
// collection assignment. The collection-to-library relationship mirrors the
// book-to-collection relationship: drag a collection tile onto a library item
// to add it; remove it via the edit-mode button.
import {
  createLibrary,
  renameLibrary,
  deleteLibrary,
  addCollectionToLibrary,
  removeCollectionFromLibrary,
} from './api';

export function initLibrary(): void {
  const layout = document.querySelector<HTMLElement>('.library-admin-layout');
  if (!layout) return;

  setupLibraryEditMode();
  setupCollectionDragAndDrop();
  setupRemoveCollectionFromLibrary();
}

// ── Library edit mode ──────────────────────────────────────────

function setupLibraryEditMode(): void {
  const editBtn = document.getElementById('library-edit-btn') as HTMLButtonElement | null;
  const addItem = document.getElementById('library-add-item') as HTMLElement | null;

  if (!editBtn) return;

  let editMode = false;

  const setEditMode = (active: boolean): void => {
    editMode = active;
    editBtn.classList.toggle('active', active);
    document.querySelectorAll<HTMLElement>('.library-delete-btn').forEach(btn => {
      btn.hidden = !active;
    });
    document.querySelectorAll<HTMLElement>('.collection-tile-remove-btn').forEach(btn => {
      btn.hidden = !active;
    });
    if (addItem) addItem.hidden = !active;
  };

  editBtn.addEventListener('click', () => setEditMode(!editMode));

  // Rename library on link click in edit mode.
  document.querySelectorAll<HTMLElement>('.library-item').forEach(item => {
    const libraryID = item.dataset.libraryId ?? '';
    if (!libraryID || item.dataset.isCentral === 'true') return; // Central Library cannot be renamed

    const link = item.querySelector<HTMLElement>('.library-link');
    link?.addEventListener('click', (e: Event) => {
      if (!editMode) return;
      e.preventDefault();
      const nameEl = item.querySelector<HTMLElement>('.library-item-name');
      if (!nameEl) return;
      startRenameLibrary(libraryID, nameEl);
    });
  });

  // Delete library buttons.
  document.querySelectorAll<HTMLButtonElement>('.library-delete-btn').forEach(btn => {
    btn.addEventListener('click', async (e: Event) => {
      e.preventDefault();
      e.stopPropagation();
      const item = btn.closest<HTMLElement>('.library-item');
      if (!item) return;
      const libraryID = item.dataset.libraryId ?? '';
      if (!libraryID) return;
      try {
        await deleteLibrary(libraryID);
        window.location.href = '/libraries/all';
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err);
        alert(`Could not delete library: ${msg}`);
      }
    });
  });

  // Create new library.
  addItem?.addEventListener('click', () => startCreateLibrary(addItem));
}

async function startRenameLibrary(libraryID: string, nameEl: HTMLElement): Promise<void> {
  const currentName = nameEl.textContent ?? '';

  const input = document.createElement('input');
  input.type = 'text';
  input.value = currentName;
  input.className = 'library-rename-input';
  nameEl.replaceWith(input);
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
        await renameLibrary(libraryID, newName);
        nameEl.textContent = newName;
      } catch (err) {
        console.error(err);
      }
    }
    input.replaceWith(nameEl);
  };

  input.addEventListener('blur', finish);
  input.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Enter') { e.preventDefault(); input.blur(); }
    if (e.key === 'Escape') { cancelled = true; input.blur(); }
  });
}

async function startCreateLibrary(addItem: HTMLElement): Promise<void> {
  const label = addItem.querySelector<HTMLElement>('.library-add-label');
  if (!label) return;

  const input = document.createElement('input');
  input.type = 'text';
  input.className = 'library-new-input';
  input.placeholder = 'Library name';
  label.replaceWith(input);
  input.focus();

  let finishing = false;

  const finish = async (): Promise<void> => {
    if (finishing) return;
    finishing = true;
    const name = input.value.trim();
    if (name) {
      try {
        const result = await createLibrary(name);
        window.location.href = `/libraries/${result.id}`;
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

// ── Drag-and-drop: collection tiles → library items ────────────

function setupCollectionDragAndDrop(): void {
  document.querySelectorAll<HTMLElement>('.collection-tile[data-collection-id]').forEach(tile => {
    tile.addEventListener('dragstart', (e: DragEvent) => {
      e.dataTransfer!.setData('text/plain', tile.dataset.collectionId!);
      e.dataTransfer!.effectAllowed = 'copy';
      tile.classList.add('dragging');
    });
    tile.addEventListener('dragend', () => {
      tile.classList.remove('dragging');
      document.querySelectorAll<HTMLElement>('.library-drop-zone.drag-over').forEach(z => {
        z.classList.remove('drag-over');
      });
    });
  });

  document.querySelectorAll<HTMLElement>('.library-drop-zone').forEach(zone => {
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
      const collectionId = e.dataTransfer!.getData('text/plain');
      const libraryId    = zone.dataset.libraryId!;
      if (collectionId && libraryId) {
        handleCollectionDrop(zone, libraryId, collectionId);
      }
    });
  });
}

async function handleCollectionDrop(
  zone: HTMLElement,
  libraryId: string,
  collectionId: string,
): Promise<void> {
  try {
    const { added } = await addCollectionToLibrary(libraryId, collectionId);
    if (added) {
      const countEl = zone.querySelector<HTMLElement>('.library-item-count');
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

// ── Remove collection from library ─────────────────────────────

function setupRemoveCollectionFromLibrary(): void {
  document.querySelectorAll<HTMLButtonElement>('.collection-tile-remove-btn').forEach(btn => {
    btn.addEventListener('click', async (e: Event) => {
      e.preventDefault();
      e.stopPropagation();
      const collectionId = btn.dataset.collectionId!;
      const libraryId    = btn.dataset.libraryId!;
      if (!collectionId || !libraryId) return;
      try {
        await removeCollectionFromLibrary(libraryId, collectionId);
        btn.closest<HTMLElement>('.collection-tile')?.remove();
        const zone = document.querySelector<HTMLElement>(
          `.library-drop-zone[data-library-id="${libraryId}"]`,
        );
        const countEl = zone?.querySelector<HTMLElement>('.library-item-count');
        if (countEl) {
          const n = parseInt(countEl.textContent?.match(/\d+/)?.[0] ?? '1', 10);
          countEl.textContent = `(${Math.max(0, n - 1)})`;
        }
      } catch (err) {
        console.error(err);
      }
    });
  });
}
