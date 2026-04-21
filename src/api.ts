// Centralized API helpers.
//
// All fetch calls go through `request`, which throws an Error on non-2xx
// responses so callers get consistent error objects without repeating the
// res.ok check everywhere.

import type { PageNotePayload } from './types';

async function request(url: string, options?: RequestInit): Promise<Response> {
  const res = await fetch(url, options);
  if (!res.ok) {
    const method = options?.method ?? 'GET';
    const status = res.status;
    const statusText = res.statusText;
    throw new Error(`${method} ${url} — ${status} ${statusText}`);
  }
  return res;
}

// ── Books ─────────────────────────────────────────────────────

export async function renameBook(bookId: string, title: string): Promise<void> {
  await request(`/api/books/${bookId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title }),
  });
}

// PersonName follows the CSL convention used in folio.json.
export interface PersonName {
  family: string;
  given: string;
}

// BookMetaPayload carries all editable metadata fields for a book.
// All fields except title are optional.
export interface BookMetaPayload {
  title: string;
  type?: string;
  abstract?: string;
  language?: string;
  author?: PersonName[];
  translator?: PersonName[];
  origtitle?: string;
  edition?: string;
  volume?: string;
  series?: string;
  series_number?: string;
  publisher?: string;
  year?: string;
  note?: string;
  keywords?: string[];
  isbn?: string;
  links?: string[];
}

export async function saveBookMeta(bookId: string, payload: BookMetaPayload): Promise<void> {
  await request(`/api/books/${bookId}/meta`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

// ── Pages ─────────────────────────────────────────────────────
//
// Page endpoints use the stable integer page ID (pages.id) rather than a
// composite key. The ID is embedded in the template as data-page-id and
// remains valid across re-scans and CBZ modifications.

export async function savePageNote(pageId: number, payload: PageNotePayload): Promise<void> {
  await request(`/api/pages/${pageId}/note`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

// svg is null to clear an existing drawing.
export async function savePageDrawing(pageId: number, svg: string | null): Promise<void> {
  await request(`/api/pages/${pageId}/drawing`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ svg_drawing: svg }),
  });
}

export async function updatePageStatus(pageId: number, status: string): Promise<void> {
  await request(`/api/pages/${pageId}/status`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  });
}

// savePageNumber sets or clears the real book page number for a page.
// Pass null to clear an existing value.
export async function savePageNumber(pageId: number, pageNumber: string | null): Promise<void> {
  await request(`/api/pages/${pageId}/page-number`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ page_number: pageNumber }),
  });
}

// ── Sections ──────────────────────────────────────────────────

export interface CreateSectionPayload {
  book_id: string;
  start_page_id: number;
  end_page_id?: number;
  title: string;
  description: string;
}

export interface UpdateSectionPayload {
  start_page_id: number;
  end_page_id?: number;
  title: string;
  description: string;
  status: string;
}

export async function createSection(payload: CreateSectionPayload): Promise<{ id: number }> {
  const res = await request('/api/sections/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  return res.json();
}

export async function updateSection(id: number, payload: UpdateSectionPayload): Promise<void> {
  await request(`/api/sections/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function deleteSection(id: number): Promise<void> {
  await request(`/api/sections/${id}`, { method: 'DELETE' });
}

// ── Collections ───────────────────────────────────────────────

export interface AddToCollectionResult {
  added: boolean; // false = book was already a member
}

export async function addBookToCollection(
  collectionId: number,
  bookId: string,
): Promise<AddToCollectionResult> {
  const res = await request(`/api/collections/${collectionId}/books/${bookId}`, {
    method: 'POST',
  });
  return res.json() as Promise<AddToCollectionResult>;
}

export async function removeBookFromCollection(
  collectionId: number,
  bookId: string,
): Promise<void> {
  await request(`/api/collections/${collectionId}/books/${bookId}`, { method: 'DELETE' });
}

// libraryId defaults to Central Library (1) when not provided.
export async function createCollection(name: string, libraryId = 1): Promise<{ id: number; name: string }> {
  const res = await request('/api/collections/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, library_id: libraryId }),
  });
  return res.json();
}

export async function renameCollection(id: number, name: string): Promise<void> {
  await request(`/api/collections/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
}

export async function deleteCollection(id: number): Promise<void> {
  await request(`/api/collections/${id}`, { method: 'DELETE' });
}

// moveCollectionToLibrary reassigns a collection to a different library.
export async function moveCollectionToLibrary(collectionId: number, libraryId: number): Promise<void> {
  await request(`/api/collections/${collectionId}/library`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ library_id: libraryId }),
  });
}

// ── Libraries ─────────────────────────────────────────────────

export async function createLibrary(name: string): Promise<{ id: number; name: string }> {
  const res = await request('/api/libraries/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
  return res.json();
}

export async function renameLibrary(id: number, name: string): Promise<void> {
  await request(`/api/libraries/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
}

export async function deleteLibrary(id: number): Promise<void> {
  await request(`/api/libraries/${id}`, { method: 'DELETE' });
}
