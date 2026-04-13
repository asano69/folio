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

    const pageIdStr = card.dataset.pageId;
    const status    = btn.dataset.status;
    if (!pageIdStr || !status) return;

    const pageId = parseInt(pageIdStr, 10);
    if (isNaN(pageId)) return;

    e.preventDefault();
    e.stopPropagation();

    try {
      await updatePageStatus(pageId, status);
      applyStatus(card, status);
    } catch (err) {
      console.error('Failed to update page status:', err);
    }
  });
}

const statusClasses = ['page-card--unread', 'page-card--reading', 'page-card--read', 'page-card--skip'];

function applyStatus(card: HTMLElement, status: string): void {
  card.classList.remove(...statusClasses);
  card.classList.add(`page-card--${status}`);

  card.querySelectorAll<HTMLButtonElement>('.status-btn').forEach(btn => {
    btn.classList.toggle('status-btn--active', btn.dataset.status === status);
  });
}
