// Shared domain types used across the frontend.

export type ReadStatus = 'unread' | 'reading' | 'read' | 'skip';

// PageNotePayload carries the user-editable text annotation body for a page.
export interface PageNotePayload {
  body: string;
}

// Section is the frontend representation of a book section.
export interface Section {
  id: number;
  bookId: string;
  startPageId: number;
  endPageId: number | null;
  title: string;
  description: string;
  status: ReadStatus;
}

export interface Collection {
  id: number;
  title: string;
}
