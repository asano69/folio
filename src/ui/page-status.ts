import { updatePageStatus } from '../api';

export function initPageStatus(): void {
  const grid = document.querySelector<HTMLElement>('.pages-grid');
  if (!grid) return;

  const bookId = grid.dataset.bookId;
  if (!bookId) return;

  grid.addEventListener('click', async (e: MouseEvent) => {
    const btn = (e.target as HTMLElement).closest<HTMLButtonElement>('.status-btn');
    if (!btn) return;

    const card = btn.closest<HTMLElement>('.page-card');
    if (!card) return;

    const pageHash = card.dataset.pageHash;
    const status = btn.dataset.status;
    if (!pageHash || !status) return;

    // Prevent the click from following the page-card-link anchor.
    e.preventDefault();
    e.stopPropagation();

    try {
      await updatePageStatus(bookId, pageHash, status);
      applyStatus(card, status);
    } catch (err) {
      console.error(err);
    }
  });
}

const statusClasses = ['page-card--unread', 'page-card--reading', 'page-card--read', 'page-card--skip'];

function applyStatus(card: HTMLElement, status: string): void {
  card.classList.remove(...statusClasses);
  // unread is the default style — no class needed, but we keep it consistent.
  card.classList.add(`page-card--${status}`);

  card.querySelectorAll<HTMLButtonElement>('.status-btn').forEach(btn => {
    btn.classList.toggle('status-btn--active', btn.dataset.status === status);
  });
}
