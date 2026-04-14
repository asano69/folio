import { savePageNote } from '../api';
import type { PageNotePayload } from '../types';
import { PANE_EVENT_EDIT_OPEN, PANE_EVENT_DRAW_OPEN } from './pane-events';

// EditorSnapshot captures the editable field values at the moment the pane
// was opened, used to restore state when the user cancels.
interface EditorSnapshot {
  noteBody: string;
}

export function initEditor(): void {
  const toggleBtn = document.getElementById('edit-toggle') as HTMLButtonElement | null;
  const pane      = document.getElementById('edit-pane')   as HTMLElement       | null;
  const closeBtn  = document.getElementById('edit-close')  as HTMLButtonElement | null;
  const backdrop  = document.getElementById('edit-backdrop') as HTMLElement     | null;

  if (!toggleBtn || !pane) return;

  // Page ID is the stable integer primary key embedded in the template.
  const pageIdStr = pane.dataset.pageId;
  if (!pageIdStr) return;
  const pageId = parseInt(pageIdStr, 10);
  if (isNaN(pageId)) return;

  const bodyTextarea = document.getElementById('edit-body')   as HTMLTextAreaElement | null;
  const saveBtn      = document.getElementById('edit-save')   as HTMLButtonElement   | null;
  const cancelBtn    = document.getElementById('edit-cancel') as HTMLButtonElement   | null;

  let snapshot = captureValues();

  const open = (): void => {
    snapshot = captureValues();
    // Notify other panes (e.g. draw pane) so they can close.
    document.dispatchEvent(new CustomEvent(PANE_EVENT_EDIT_OPEN));
    pane.classList.add('open');
    backdrop?.classList.add('visible');
    toggleBtn.classList.add('active');
    bodyTextarea?.focus();
  };

  const close = (): void => {
    pane.classList.remove('open');
    backdrop?.classList.remove('visible');
    toggleBtn.classList.remove('active');
  };

  toggleBtn.addEventListener('click', () => {
    if (pane.classList.contains('open')) {
      restoreSnapshot();
      close();
    } else {
      open();
    }
  });

  // X button and backdrop both discard unsaved changes.
  closeBtn?.addEventListener('click', () => { restoreSnapshot(); close(); });
  backdrop?.addEventListener('click', () => { restoreSnapshot(); close(); });

  saveBtn?.addEventListener('click',   () => { save(); });
  cancelBtn?.addEventListener('click', () => { restoreSnapshot(); close(); });

  function captureValues(): EditorSnapshot {
    return {
      noteBody: bodyTextarea?.value ?? '',
    };
  }

  function restoreSnapshot(): void {
    if (bodyTextarea) bodyTextarea.value = snapshot.noteBody;
  }

  async function save(): Promise<void> {
    if (!saveBtn) return;
    saveBtn.disabled = true;
    try {
      const current = captureValues();
      const payload: PageNotePayload = { body: current.noteBody };
      await savePageNote(pageId, payload);
      snapshot = current;
      updateNoteDisplay(current.noteBody);
      close();
    } catch (err) {
      console.error(err);
    } finally {
      saveBtn.disabled = false;
    }
  }

  function updateNoteDisplay(body: string): void {
    const noteEl   = document.getElementById('page-note') as HTMLElement | null;
    const noteBody = document.getElementById('note-body') as HTMLElement | null;
    if (noteEl && noteBody) {
      noteBody.textContent = body;
      noteEl.hidden = !body;
    }
  }

  // Close when the draw pane is opened.
  document.addEventListener(PANE_EVENT_DRAW_OPEN, close);
}
