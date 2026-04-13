// Shared domain types used across the frontend.

export type ReadStatus = 'unread' | 'reading' | 'read' | 'skip';

// PageNotePayload carries the user-editable text annotation fields for a page.
export interface PageNotePayload {
  title: string;
  body:  string;
}

// PageSectionPayload marks or unmarks a page as a section start.
// When enabled is false, the section entry is removed.
export interface PageSectionPayload {
  title:   string;
  enabled: boolean;
}

export interface Collection {
  id: number;
  title: string;
}
