// Shared domain types used across the frontend.

export type ReadStatus = 'unread' | 'reading' | 'read' | 'skip';

// PageEditPayload carries the user-editable fields for a single page.
// title and attribute are stored on the pages row; body is stored in notes.
export interface PageEditPayload {
  title: string;
  attribute: string;
  body: string;
}

export interface Collection {
  id: number;
  title: string;
}
