// Pane cross-communication event names.
//
// Using typed constants instead of raw string literals prevents silent failures
// caused by typos that TypeScript cannot detect in addEventListener / dispatchEvent calls.

export const PANE_EVENT_EDIT_OPEN = 'folio:edit-pane-open' as const;
export const PANE_EVENT_DRAW_OPEN = 'folio:draw-pane-open' as const;
