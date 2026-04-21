// Library admin page interactions: CRUD for libraries and collection assignment.
import {
  createLibrary,
  renameLibrary,
  deleteLibrary,
  moveCollectionToLibrary,
} from '../api';

export function initLibrary(): void {
  const layout = document.querySelector<HTMLElement>('.library-admin-layout');
  if (!layout) return;

  setupLibraryEditMode();
  setupCollectionMoveSelects();
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
    document.querySelectorAll<HTMLElement>('.collection-tile-move').forEach(el => {
      el.hidden = !active;
    });
    if (addItem) addItem.hidden = !active;
  };

  editBtn.addEventListener('click', () => setEditMode(!editMode));

  // Rename library on link click in edit mode.
  document.querySelectorAll<HTMLElement>('.library-item').forEach(item => {
    const libraryID = parseInt(item.dataset.libraryId ?? '0', 10);
    if (isNaN(libraryID) || libraryID === 1) return; // Central Library cannot be renamed

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
      const libraryID = parseInt(item.dataset.libraryId ?? '0', 10);
      if (isNaN(libraryID) || libraryID === 0) return;
      try {
        await deleteLibrary(libraryID);
        window.location.href = '/library';
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err);
        alert(`Could not delete library: ${msg}`);
      }
    });
  });

  // Create new library.
  addItem?.addEventListener('click', () => startCreateLibrary(addItem));
}

async function startRenameLibrary(libraryID: number, nameEl: HTMLElement): Promise<void> {
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
        window.location.href = `/library?lib=${result.id}`;
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

// ── Collection move selects ────────────────────────────────────

// Each collection tile has a <select> to move it to a different library.
// Changing the select immediately moves the collection and reloads the page.
function setupCollectionMoveSelects(): void {
  document.querySelectorAll<HTMLSelectElement>('.collection-library-select').forEach(sel => {
    sel.addEventListener('change', async () => {
      const collectionID = parseInt(sel.dataset.collectionId ?? '0', 10);
      const targetLibraryID = parseInt(sel.value, 10);
      if (isNaN(collectionID) || isNaN(targetLibraryID)) return;
      try {
        await moveCollectionToLibrary(collectionID, targetLibraryID);
        // Reload to show the tile has moved; navigate to the target library.
        window.location.href = `/library?lib=${targetLibraryID}`;
      } catch (err) {
        console.error('Failed to move collection:', err);
        // Restore original selection on error.
        const tile = sel.closest<HTMLElement>('.collection-tile');
        if (tile) {
          const originalLibraryID = tile.dataset.libraryId ?? '';
          sel.value = originalLibraryID;
        }
      }
    });
  });
}
