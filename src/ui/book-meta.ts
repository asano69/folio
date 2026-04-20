// Metadata editor for the bibliography page.
//
// Reads initial values from the server-rendered form inputs and from
// <script type="application/json"> tags (for array fields). On save, collects
// all field values and sends a single PUT /api/books/{id}/meta request.

import { saveBookMeta } from '../api';
import type { PersonName } from '../api';

// parsePersonNameLines parses a textarea value where each non-empty line is
// "Family, Given". Lines with no comma are treated as family-name-only entries.
function parsePersonNameLines(text: string): PersonName[] {
  return text
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0)
    .map(line => {
      const commaIdx = line.indexOf(',');
      if (commaIdx === -1) {
        return { family: line, given: '' };
      }
      return {
        family: line.slice(0, commaIdx).trim(),
        given: line.slice(commaIdx + 1).trim(),
      };
    });
}

// formatPersonNamesToLines converts a PersonName array into one "Family, Given"
// line per person, for display in a textarea.
function formatPersonNamesToLines(names: PersonName[]): string {
  return names.map(n => (n.given ? `${n.family}, ${n.given}` : n.family)).join('\n');
}

// parseStringLines splits a textarea value into a trimmed, non-empty string array.
function parseStringLines(text: string): string[] {
  return text
    .split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0);
}

// readScriptJSON reads and parses the content of a <script type="application/json">
// element. Returns the fallback value if the element is missing or unparseable.
function readScriptJSON<T>(elementId: string, fallback: T): T {
  const el = document.getElementById(elementId);
  if (!el) return fallback;
  try {
    const parsed = JSON.parse(el.textContent ?? '') as T | null;
    return parsed ?? fallback;
  } catch {
    return fallback;
  }
}

export function initBookMeta(): void {
  const form = document.getElementById('book-meta-form') as HTMLElement | null;
  if (!form) return;

  const bookId = form.dataset.bookId;
  if (!bookId) return;

  const saveBtn = document.getElementById('book-meta-save') as HTMLButtonElement | null;
  const statusEl = document.getElementById('book-meta-status') as HTMLElement | null;

  // ── Populate array fields from JSON script tags ────────────────────────────
  // These fields cannot be safely embedded in HTML attribute values, so the
  // server writes them into <script type="application/json"> elements instead.

  const authorTextarea = document.getElementById('meta-author') as HTMLTextAreaElement | null;
  const translatorTextarea = document.getElementById('meta-translator') as HTMLTextAreaElement | null;
  const keywordsTextarea = document.getElementById('meta-keywords') as HTMLTextAreaElement | null;
  const linksTextarea = document.getElementById('meta-links') as HTMLTextAreaElement | null;

  if (authorTextarea) {
    const authors = readScriptJSON<PersonName[]>('book-author-data', []);
    authorTextarea.value = formatPersonNamesToLines(authors);
  }
  if (translatorTextarea) {
    const translators = readScriptJSON<PersonName[]>('book-translator-data', []);
    translatorTextarea.value = formatPersonNamesToLines(translators);
  }
  if (keywordsTextarea) {
    const keywords = readScriptJSON<string[]>('book-keywords-data', []);
    keywordsTextarea.value = keywords.join('\n');
  }
  if (linksTextarea) {
    const links = readScriptJSON<string[]>('book-links-data', []);
    linksTextarea.value = links.join('\n');
  }

  // ── Save handler ───────────────────────────────────────────────────────────

  saveBtn?.addEventListener('click', async () => {
    if (!saveBtn) return;
    saveBtn.disabled = true;
    if (statusEl) statusEl.hidden = true;

    const title = (document.getElementById('meta-title') as HTMLInputElement | null)?.value.trim() ?? '';
    if (!title) {
      showStatus(statusEl, 'Title cannot be empty.', 'error');
      saveBtn.disabled = false;
      return;
    }

    const payload = {
      title,
      type:          fieldValue('meta-type'),
      abstract:      fieldValue('meta-abstract'),
      language:      fieldValue('meta-language'),
      author:        parsePersonNameLines(authorTextarea?.value ?? ''),
      translator:    parsePersonNameLines(translatorTextarea?.value ?? ''),
      origtitle:     fieldValue('meta-origtitle'),
      edition:       fieldValue('meta-edition'),
      volume:        fieldValue('meta-volume'),
      series:        fieldValue('meta-series'),
      series_number: fieldValue('meta-series-number'),
      publisher:     fieldValue('meta-publisher'),
      year:          fieldValue('meta-year'),
      note:          fieldValue('meta-note'),
      keywords:      parseStringLines(keywordsTextarea?.value ?? ''),
      isbn:          fieldValue('meta-isbn'),
      links:         parseStringLines(linksTextarea?.value ?? ''),
    };

    try {
      await saveBookMeta(bookId, payload);
      showStatus(statusEl, '✓ Saved', 'success');
      // Update the page <h2> title to reflect any rename.
      const h2 = document.querySelector<HTMLElement>('.library-header h2');
      if (h2) h2.textContent = title;
    } catch (err) {
      console.error('Failed to save book metadata:', err);
      showStatus(statusEl, 'Save failed.', 'error');
    } finally {
      saveBtn.disabled = false;
    }
  });
}

// fieldValue reads the trimmed value of an input or textarea by element ID.
function fieldValue(id: string): string {
  const el = document.getElementById(id) as HTMLInputElement | HTMLTextAreaElement | null;
  return el?.value.trim() ?? '';
}

// showStatus displays a temporary status message next to the save button.
function showStatus(el: HTMLElement | null, message: string, kind: 'success' | 'error'): void {
  if (!el) return;
  el.textContent = message;
  el.className = `bibliographic-meta-status bibliographic-meta-status--${kind}`;
  el.hidden = false;
  setTimeout(() => {
    el.hidden = true;
  }, 3000);
}
