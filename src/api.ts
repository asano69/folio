// Centralized API helpers.
//
// All fetch calls go through `request`, which throws an Error on non-2xx
// responses so callers get consistent error objects without repeating the
// res.ok check everywhere.

import type { NotePayload } from './types';

async function request(url: string, options?: RequestInit): Promise<Response> {
  const res = await fetch(url, options);
  if (!res.ok) {
    throw new Error(`${options?.method ?? 'GET'} ${url} — ${res.status} ${res.statusText}`);
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

// ── Pages ─────────────────────────────────────────────────────

export async function saveNote(bookId: string, pageHash: string, payload: NotePayload): Promise<void> {
  await request(`/api/pages/${bookId}/${pageHash}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

// svg is null to clear an existing drawing.
export async function saveDrawing(bookId: string, pageHash: string, svg: string | null): Promise<void> {
  await request(`/api/pages/${bookId}/${pageHash}/drawing`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ svg_drawing: svg }),
  });
}

// ── Collections ───────────────────────────────────────────────

export interface AddToCollectionResult {
  added: boolean; // false = book was already a member
}

export async function addBookToCollection(
  collectionId: string,
  bookId: string,
): Promise<AddToCollectionResult> {
  const res = await request(`/api/collections/${collectionId}/books/${bookId}`, {
    method: 'POST',
  });
  return res.json() as Promise<AddToCollectionResult>;
}

export async function removeBookFromCollection(
  collectionId: string,
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

export async function renameCollection(id: string, title: string): Promise<void> {
  await request(`/api/collections/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title }),
  });
}

export async function deleteCollection(id: string): Promise<void> {
  await request(`/api/collections/${id}`, { method: 'DELETE' });
}
