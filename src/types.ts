// Shared domain types used across the frontend.

export type ReadStatus = 'unread' | 'reading' | 'read' | 'skip';

export interface NotePayload {
  title: string;
  attribute: string;
  body: string;
}

export interface Collection {
  id: number;
  title: string;
}
