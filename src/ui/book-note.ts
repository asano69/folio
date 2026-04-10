import { saveBookNote } from '../api';

export function initBookNote(): void {
  const textarea = document.getElementById('book-note-body') as HTMLTextAreaElement | null;
  const saveBtn = document.getElementById('book-note-save') as HTMLButtonElement | null;

  if (!textarea || !saveBtn) return;

  const bookId = textarea.dataset.bookId;
  if (!bookId) return;

  saveBtn.addEventListener('click', async () => {
    saveBtn.disabled = true;
    try {
      await saveBookNote(bookId, textarea.value);
    } catch (err) {
      console.error(err);
    } finally {
      saveBtn.disabled = false;
    }
  });
}
