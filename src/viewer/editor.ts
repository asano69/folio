import { savePageNote, savePageSection } from '../api';
import type { PageNotePayload, PageSectionPayload } from '../types';
import { PANE_EVENT_EDIT_OPEN, PANE_EVENT_DRAW_OPEN } from './pane-events';

// EditorSnapshot captures all editable field values at a point in time,
// used to restore the pane to its original state when the user cancels.
interface EditorSnapshot {
  noteTitle:    string;
  noteBody:     string;
  isSection:    boolean;
  sectionTitle: string;
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

  const noteTitleInput    = document.getElementById('edit-note-title')    as HTMLInputElement   | null;
  const bodyTextarea      = document.getElementById('edit-body')          as HTMLTextAreaElement | null;
  const sectionToggle     = document.getElementById('edit-section-toggle') as HTMLInputElement  | null;
  const sectionTitleInput = document.getElementById('edit-section-title') as HTMLInputElement   | null;
  const sectionTitleField = document.getElementById('edit-section-title-field') as HTMLElement  | null;
  const saveBtn           = document.getElementById('edit-save')          as HTMLButtonElement  | null;
  const cancelBtn         = document.getElementById('edit-cancel')        as HTMLButtonElement  | null;

  // Show or hide the section title input based on the toggle state.
  sectionToggle?.addEventListener('change', () => {
    if (sectionTitleField) sectionTitleField.hidden = !sectionToggle.checked;
  });

  // Snapshot of field values at the moment the pane was opened.
  let snapshot = captureValues();

  const open = (): void => {
    snapshot = captureValues();
    // Notify other panes (e.g. draw pane) so they can close.
    document.dispatchEvent(new CustomEvent(PANE_EVENT_EDIT_OPEN));
    pane.classList.add('open');
    backdrop?.classList.add('visible');
    toggleBtn.classList.add('active');
    noteTitleInput?.focus();
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

  function captureValues(): EditorSnapshot {
    return {
      noteTitle:    noteTitleInput?.value    ?? '',
      noteBody:     bodyTextarea?.value      ?? '',
      isSection:    sectionToggle?.checked   ?? false,
      sectionTitle: sectionTitleInput?.value ?? '',
    };
  }

  function restoreSnapshot(): void {
    if (noteTitleInput)    noteTitleInput.value    = snapshot.noteTitle;
    if (bodyTextarea)      bodyTextarea.value      = snapshot.noteBody;
    if (sectionToggle)     sectionToggle.checked   = snapshot.isSection;
    if (sectionTitleInput) sectionTitleInput.value = snapshot.sectionTitle;
    if (sectionTitleField) sectionTitleField.hidden = !snapshot.isSection;
  }

  async function save(): Promise<void> {
    if (!saveBtn) return;
    saveBtn.disabled = true;
    try {
      const current = captureValues();

      const notePayload: PageNotePayload = {
        title: current.noteTitle,
        body:  current.noteBody,
      };
      const sectionPayload: PageSectionPayload = {
        title:   current.sectionTitle,
        enabled: current.isSection,
      };

      await savePageNote(pageId, notePayload);
      await savePageSection(pageId, sectionPayload);

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
