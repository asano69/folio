// Centralized API helpers.
//
// All fetch calls go through `request`, which throws an Error on non-2xx
// responses so callers get consistent error objects without repeating the
// res.ok check everywhere.

import type { PageEditPayload } from './types';

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

export async function saveBookNote(bookId: string, body: string): Promise<void> {
  await request(`/api/books/${bookId}/note`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ body }),
  });
}

// ── Pages ─────────────────────────────────────────────────────
//
// Page endpoints use the stable integer page ID (pages.id) rather than a
// (bookID, pageHash) pair. The ID is embedded in the template as data-page-id
// and remains valid across re-scans and CBZ modifications.

export async function savePageEdit(pageId: number, payload: PageEditPayload): Promise<void> {
  await request(`/api/pages/${pageId}`, {
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

export async function createCollection(title: string): Promise<{ id: number; title: string }> {
  const res = await request('/api/collections/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title }),
  });
  return res.json();
}

export async function renameCollection(id: number, title: string): Promise<void> {
  await request(`/api/collections/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title }),
  });
}

export async function deleteCollection(id: number): Promise<void> {
  await request(`/api/collections/${id}`, { method: 'DELETE' });
}
