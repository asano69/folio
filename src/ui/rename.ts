// Inline rename handler for book titles on the library page.
import { renameBook } from '../api';

export function initRename(): void {
  document.querySelectorAll<HTMLButtonElement>('.rename-btn').forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      const bookId = btn.dataset.bookId;
      if (!bookId) return;

      // The h3 is a direct flex child of .book-info; replacing it with an
      // input keeps the input as a flex child, avoiding the HTML restriction
      // against placing interactive content inside <a>.
      const titleEl = document.querySelector<HTMLElement>(
        `.book-title[data-book-id="${bookId}"]`
      );
      if (!titleEl) return;

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
