import { savePageEdit } from '../api';
import type { PageEditPayload } from '../types';
import { PANE_EVENT_EDIT_OPEN, PANE_EVENT_DRAW_OPEN } from './pane-events';

export function initEditor(): void {
  const toggleBtn = document.getElementById('edit-toggle') as HTMLButtonElement | null;
  const pane = document.getElementById('edit-pane') as HTMLElement | null;
  const closeBtn = document.getElementById('edit-close') as HTMLButtonElement | null;
  const backdrop = document.getElementById('edit-backdrop') as HTMLElement | null;

  if (!toggleBtn || !pane) return;

  // Page ID is the stable integer primary key embedded in the template.
  const pageIdStr = pane.dataset.pageId;
  if (!pageIdStr) return;
  const pageId = parseInt(pageIdStr, 10);
  if (isNaN(pageId)) return;

  const titleInput     = document.getElementById('edit-title')     as HTMLInputElement;
  const attributeSelect = document.getElementById('edit-attribute') as HTMLSelectElement;
  const bodyTextarea   = document.getElementById('edit-body')      as HTMLTextAreaElement;
  const saveBtn        = document.getElementById('edit-save')      as HTMLButtonElement;
  const cancelBtn      = document.getElementById('edit-cancel')    as HTMLButtonElement;

  // Snapshot of field values at the moment the pane was opened.
  let snapshot = captureValues();

  const open = (): void => {
    snapshot = captureValues();
    // Notify other panes (e.g. draw pane) so they can close.
    document.dispatchEvent(new CustomEvent(PANE_EVENT_EDIT_OPEN));
    pane.classList.add('open');
    backdrop?.classList.add('visible');
    toggleBtn.classList.add('active');
    titleInput?.focus();
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

  saveBtn?.addEventListener('click', () => { save(); });

  cancelBtn?.addEventListener('click', () => {
    restoreSnapshot();
    close();
  });

  function captureValues(): PageEditPayload {
    return {
      title:     titleInput?.value ?? '',
      attribute: attributeSelect?.value ?? '',
      body:      bodyTextarea?.value ?? '',
    };
  }

  function restoreSnapshot(): void {
    if (titleInput)      titleInput.value      = snapshot.title;
    if (attributeSelect) attributeSelect.value = snapshot.attribute;
    if (bodyTextarea)    bodyTextarea.value     = snapshot.body;
  }

  async function save(): Promise<void> {
    if (!saveBtn) return;
    saveBtn.disabled = true;
    try {
      const payload = captureValues();
      await savePageEdit(pageId, payload);
      snapshot = payload;
      updateNoteDisplay(payload.body);
      close();
    } catch (err) {
      console.error(err);
    } finally {
      saveBtn.disabled = false;
    }
  }

  function updateNoteDisplay(body: string): void {
    const noteEl   = document.getElementById('page-note')  as HTMLElement | null;
    const noteBody = document.getElementById('note-body')  as HTMLElement | null;
    if (noteEl && noteBody) {
      noteBody.textContent = body;
      noteEl.hidden = !body;
    }
  }

  // Close when the draw pane is opened.
  document.addEventListener(PANE_EVENT_DRAW_OPEN, close);
}
