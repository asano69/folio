// Inline rename handler for book titles on the library page.
// Rename is activated globally via the Edit button rather than per-card icons.
import { renameBook } from '../api';

let editMode = false;

export function initRename(): void {
  const editBtn = document.getElementById('shelf-edit-btn') as HTMLButtonElement | null;
  if (!editBtn) return;

  const booksMain = document.querySelector<HTMLElement>('.books-main');

  editBtn.addEventListener('click', () => {
    editMode = !editMode;
    editBtn.classList.toggle('active', editMode);
    booksMain?.classList.toggle('shelf--edit', editMode);
  });

  // Clicking a book title in edit mode starts an inline rename.
  document.querySelectorAll<HTMLElement>('.book-title[data-book-id]').forEach(titleEl => {
    titleEl.addEventListener('click', (e) => {
      if (!editMode) return;
      e.preventDefault();
      const bookId = titleEl.dataset.bookId;
      if (!bookId) return;
      startRename(titleEl, bookId);
    });
  });
}

async function startRename(titleEl: HTMLElement, bookId: string): Promise<void> {
  // Read the visible text from the inner <a> if present, otherwise the element itself.
  const linkEl = titleEl.querySelector<HTMLAnchorElement>('a');
  const currentTitle = (linkEl ?? titleEl).textContent ?? '';

  const input = document.createElement('input');
  input.type = 'text';
  input.value = currentTitle;
  input.className = 'rename-input';

  // Replace the h3 with the input; input becomes a flex child of .book-info.
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
        await renameBook(bookId, newTitle);
        // Update the link text inside the h3 without touching the href.
        (linkEl ?? titleEl).textContent = newTitle;
      } catch (err) {
        console.error(err);
      }
    }

    input.replaceWith(titleEl);
  };

  input.addEventListener('blur', finish);
  input.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      input.blur();
    } else if (e.key === 'Escape') {
      cancelled = true;
      input.blur();
    }
  });
}
