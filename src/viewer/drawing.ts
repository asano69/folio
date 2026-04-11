// src/viewer/drawing.ts

import { saveDrawing } from '../api';
import { PANE_EVENT_DRAW_OPEN, PANE_EVENT_EDIT_OPEN } from './pane-events';

// ── Type definitions ───────────────────────────────────────────

interface PenSettings {
  color:   string;
  opacity: number; // 0–1
  size:    number;
}

interface EraserSettings {
  size: number;
}

// A history entry is either an ink stroke that was added, or a set of paths
// that were removed by the eraser.
type HistoryEntry =
  | { kind: 'add';   element: SVGPathElement }
  | { kind: 'erase'; removed: SVGPathElement[] };

// ── Drawing state management ───────────────────────────────────
// Tracks whether the drawing has unsaved changes and save status.

interface DrawingState {
  isDirty: boolean;
  isSaving: boolean;
  lastSavedSVG: string | null;
  unsavedChanges: number; // Count of strokes since last save
}

// ── SVG Validation ─────────────────────────────────────────────
// validateSVG checks that the SVG markup is well-formed and parseable.
// Returns true if valid, false otherwise.

function validateSVG(svg: string): boolean {
  try {
    const parser = new DOMParser();
    const doc = parser.parseFromString(
      `<svg xmlns="http://www.w3.org/2000/svg">${svg}</svg>`,
      'image/svg+xml',
    );

    // DOMParser silently creates a <parsererror> element on failure.
    // Check for this element to detect parse errors.
    const hasParseError = doc.querySelector('parsererror') !== null;
    return !hasParseError;
  } catch {
    return false;
  }
}

// ── Main initialization ────────────────────────────────────────

export function initDrawing(): void {
  const toggleBtn = document.getElementById('draw-toggle')  as HTMLButtonElement | null;
  const pane      = document.getElementById('draw-pane')    as HTMLElement       | null;
  const closeBtn  = document.getElementById('draw-close')   as HTMLButtonElement | null;
  const backdrop  = document.getElementById('draw-backdrop') as HTMLElement      | null;
  const image     = document.getElementById('page-image')   as HTMLImageElement  | null;
  const overlay   = document.getElementById('drawing-overlay') as SVGSVGElement  | null;

  if (!toggleBtn || !pane || !image || !overlay) return;

  const bookId   = overlay.dataset.bookId   ?? '';
  const pageHash = overlay.dataset.pageHash ?? '';

  // Drawing requires a page hash (computed by `folio hash` or `folio scan`).
  if (!bookId || !pageHash) {
    toggleBtn.disabled = true;
    toggleBtn.title    = 'Drawing unavailable: page hash not computed yet';
    return;
  }

  const inkLayer = overlay.querySelector<SVGGElement>('#drawing-ink')!;

  // ── State ──────────────────────────────────────────────────────────────────

  const pen:    PenSettings    = { color: '#e74c3c', opacity: 1, size: 4 };
  const eraser: EraserSettings = { size: 20 };
  let activeTool: 'ink' | 'erase' = 'ink';

  const undoStack: HistoryEntry[] = [];
  const redoStack: HistoryEntry[] = [];

  const state: DrawingState = {
    isDirty: false,
    isSaving: false,
    lastSavedSVG: null,
    unsavedChanges: 0,
  };

  // ── SVG viewBox ────────────────────────────────────────────────────────

  const applyViewBox = (): void => {
    if (image.naturalWidth && image.naturalHeight) {
      overlay.setAttribute('viewBox', `0 0 ${image.naturalWidth} ${image.naturalHeight}`);
    }
  };

  if (image.complete && image.naturalWidth) {
    applyViewBox();
  } else {
    image.addEventListener('load', applyViewBox, { once: true });
  }

  // ── Transform sync ─────────────────────────────────────────────────────────
  // Mirror any zoom/pan transform applied to the image onto the SVG overlay
  // so both move together. transform-origin defaults to 'center' on both.

  const syncTransform = (): void => {
    overlay.style.transform       = image.style.transform;
    overlay.style.transformOrigin = 'center';
  };

  new MutationObserver(syncTransform).observe(image, {
    attributes:      true,
    attributeFilter: ['style'],
  });

  // ── Restore existing drawing ───────────────────────────────────────────────

  const existingData = overlay.dataset.drawing;
  if (existingData) {
    if (!validateSVG(existingData)) {
      console.error('Stored SVG is malformed. Starting with empty canvas.');
    } else {
      restoreDrawing(existingData, inkLayer);
      state.lastSavedSVG = existingData;
    }
  }

  // ── Pane open / close ──────────────────────────────────────────────────────

  const setDrawingMode = (active: boolean): void => {
    overlay.style.pointerEvents = active ? 'all'       : 'none';
    overlay.style.cursor        = active ? 'crosshair' : '';
    overlay.style.touchAction   = active ? 'none'      : '';
  };

  const openPane = (): void => {
    document.dispatchEvent(new CustomEvent(PANE_EVENT_DRAW_OPEN));
    pane.classList.add('open');
    // No backdrop: the user must be able to click the image to draw.
    toggleBtn.classList.add('active');
    setDrawingMode(true);
  };

  const closePane = (): void => {
    // Warn user if there are unsaved changes.
    if (state.isDirty && state.unsavedChanges > 0) {
      const confirmed = confirm(
        'You have unsaved drawing changes. Close anyway?'
      );
      if (!confirmed) return;
    }
    pane.classList.remove('open');
    toggleBtn.classList.remove('active');
    setDrawingMode(false);
  };

  toggleBtn.addEventListener('click', () => {
    pane.classList.contains('open') ? closePane() : openPane();
  });
  closeBtn?.addEventListener('click',  closePane);
  backdrop?.addEventListener('click',  closePane);

  // Close when the edit pane is opened.
  document.addEventListener(PANE_EVENT_EDIT_OPEN, closePane);

  // ── Tool UI ────────────────────────────────────────────────────────────────

  const penBtn       = document.getElementById('draw-tool-pen')    as HTMLButtonElement | null;
  const eraserBtn    = document.getElementById('draw-tool-eraser') as HTMLButtonElement | null;
  const colorInput   = document.getElementById('draw-color')       as HTMLInputElement  | null;
  const opacityInput = document.getElementById('draw-opacity')     as HTMLInputElement  | null;
  const sizeInput    = document.getElementById('draw-size')        as HTMLInputElement  | null;
  const opacityVal   = document.getElementById('draw-opacity-val');
  const sizeVal      = document.getElementById('draw-size-val');
  const colorField   = document.getElementById('draw-color-field');
  const opacityField = document.getElementById('draw-opacity-field');
  const saveBtn      = document.getElementById('draw-save')        as HTMLButtonElement | null;

  const syncToolUI = (): void => {
    const isPen = activeTool === 'ink';
    penBtn?.classList.toggle('active', isPen);
    eraserBtn?.classList.toggle('active', !isPen);
    if (colorField)   colorField.hidden   = !isPen;
    if (opacityField) opacityField.hidden = !isPen;
    if (sizeInput) {
      sizeInput.max   = isPen ? '50' : '80';
      sizeInput.value = String(isPen ? pen.size : eraser.size);
    }
    if (sizeVal) sizeVal.textContent = `${sizeInput?.value ?? '4'}px`;
  };

  penBtn?.addEventListener('click',   () => { activeTool = 'ink';   syncToolUI(); });
  eraserBtn?.addEventListener('click', () => { activeTool = 'erase'; syncToolUI(); });

  colorInput?.addEventListener('input', () => {
    pen.color = colorInput.value;
  });

  opacityInput?.addEventListener('input', () => {
    pen.opacity = parseInt(opacityInput.value, 10) / 100;
    if (opacityVal) opacityVal.textContent = `${opacityInput.value}%`;
  });

  sizeInput?.addEventListener('input', () => {
    const v = parseInt(sizeInput.value, 10);
    if (activeTool === 'ink') pen.size = v; else eraser.size = v;
    if (sizeVal) sizeVal.textContent = `${v}px`;
  });

  syncToolUI();

  // ── Save with validation and error handling ────────────────────────────────

  saveBtn?.addEventListener('click', async () => {
    if (!saveBtn) return;
    if (state.isSaving) return; // Prevent duplicate submissions.

    saveBtn.disabled = true;
    state.isSaving = true;

    try {
      const svg = serializeDrawing(inkLayer);

      // Validate the SVG before sending.
      if (svg !== null && !validateSVG(svg)) {
        throw new Error('Drawing contains invalid SVG markup. Cannot save.');
      }

      await saveDrawing(bookId, pageHash, svg);

      // Update state after successful save.
      state.lastSavedSVG = svg;
      state.isDirty = false;
      state.unsavedChanges = 0;

      // Provide visual feedback.
      saveBtn.textContent = '✓ Saved';
      setTimeout(() => {
        saveBtn.textContent = 'Save';
      }, 2000);
    } catch (err) {
      console.error('Failed to save drawing:', err);

      // Rollback: restore the last successfully saved state.
      if (state.lastSavedSVG !== null) {
        inkLayer.innerHTML = '';
        restoreDrawing(state.lastSavedSVG, inkLayer);
      }

      // Alert the user.
      alert(
        'Failed to save drawing. Your changes have been reverted to the last saved state.' +
        '\n\nError: ' + (err instanceof Error ? err.message : String(err))
      );
    } finally {
      saveBtn.disabled = false;
      state.isSaving = false;
    }
  });

  // ── Undo / redo ────────────────────────────────────────────────────────────

  const markDirty = (): void => {
    state.isDirty = true;
    state.unsavedChanges++;
  };

  // Undo/redo is dispatched by the centralized keyboard manager in
  // src/keyboard/init.ts via folio:draw-undo and folio:draw-redo custom events.
  // A local keydown listener is intentionally absent here to avoid double-firing:
  // KeyBindingManager calls e.preventDefault() but not e.stopPropagation(), so
  // any additional keydown listener on document would also receive the same event,
  // causing undoEntry / redoEntry to run twice per keystroke.
  document.addEventListener('folio:draw-undo', () => {
    if (!pane.classList.contains('open')) return;
    undoEntry(undoStack, redoStack, inkLayer);
    markDirty();
  });

  document.addEventListener('folio:draw-redo', () => {
    if (!pane.classList.contains('open')) return;
    redoEntry(undoStack, redoStack, inkLayer);
    markDirty();
  });

  // ── Drawing interaction ────────────────────────────────────────────────────

  let currentPath: SVGPathElement | null = null;
  let pathData = '';
  let drawing  = false;

  // Paths removed during the current eraser drag, grouped into one history entry.
  let currentEraseRemoved: SVGPathElement[] = [];

  overlay.addEventListener('pointerdown', (e: PointerEvent) => {
    if (e.button !== 0) return;
    e.preventDefault();
    overlay.setPointerCapture(e.pointerId);

    // Any new stroke invalidates the redo history.
    redoStack.splice(0);
    markDirty();

    const pt = toSVGPoint(overlay, e.clientX, e.clientY);

    if (activeTool === 'ink') {
      currentPath = makePenPath(pen, pt.x, pt.y);
      pathData    = `M ${pt.x} ${pt.y}`;
      currentPath.setAttribute('d', pathData);
      inkLayer.appendChild(currentPath);
    } else {
      currentEraseRemoved = [];
      eraseAt(inkLayer, pt.x, pt.y, eraser.size, currentEraseRemoved);
    }

    drawing = true;
  });

  overlay.addEventListener('pointermove', (e: PointerEvent) => {
    if (!drawing) return;
    const pt = toSVGPoint(overlay, e.clientX, e.clientY);

    if (activeTool === 'ink' && currentPath) {
      pathData += ` L ${pt.x} ${pt.y}`;
      currentPath.setAttribute('d', pathData);
    } else if (activeTool === 'erase') {
      eraseAt(inkLayer, pt.x, pt.y, eraser.size, currentEraseRemoved);
    }
  });

  const endStroke = (): void => {
    if (!drawing) return;
    drawing = false;

    if (activeTool === 'ink' && currentPath) {
      undoStack.push({ kind: 'add', element: currentPath });
      currentPath = null;
      pathData    = '';
    } else if (activeTool === 'erase' && currentEraseRemoved.length > 0) {
      undoStack.push({ kind: 'erase', removed: currentEraseRemoved });
      currentEraseRemoved = [];
    }
  };

  overlay.addEventListener('pointerup',     endStroke);
  overlay.addEventListener('pointercancel', endStroke);
}

// ── Helper functions ───────────────────────────────────────────

// eraseAt removes all ink paths whose bounding box overlaps with a circle of
// the given radius centred at (x, y). Removed elements are appended to the
// `removed` array so the caller can record them for undo.
//
// Bounding-box intersection is a conservative approximation: it may match paths
// whose strokes do not visually overlap the eraser, but is fast and reliable
// across all browsers without needing getIntersectionList.
function eraseAt(
  inkLayer: SVGGElement,
  x: number,
  y: number,
  size: number,
  removed: SVGPathElement[],
): void {
  const half = size / 2;
  // Iterate a static snapshot because we mutate the live collection.
  const children = Array.from(inkLayer.children) as SVGPathElement[];
  for (const path of children) {
    const bb = path.getBBox();
    const overlaps =
      bb.x            < x + half &&
      bb.x + bb.width > x - half &&
      bb.y             < y + half &&
      bb.y + bb.height > y - half;
    if (overlaps) {
      path.remove();
      removed.push(path);
    }
  }
}

// ── Undo / redo helpers ────────────────────────────────────────

function undoEntry(
  undoStack: HistoryEntry[],
  redoStack: HistoryEntry[],
  inkLayer:  SVGGElement,
): void {
  const entry = undoStack.pop();
  if (!entry) return;
  if (entry.kind === 'add') {
    entry.element.remove();
  } else {
    // Restore removed paths. Order is preserved; they are re-appended at the
    // end of the layer, which may differ from the original Z-order when paths
    // from multiple erase operations are interleaved, but is acceptable for
    // annotation use.
    for (const el of entry.removed) {
      inkLayer.appendChild(el);
    }
  }
  redoStack.push(entry);
}

function redoEntry(
  undoStack: HistoryEntry[],
  redoStack: HistoryEntry[],
  inkLayer:  SVGGElement,
): void {
  const entry = redoStack.pop();
  if (!entry) return;
  if (entry.kind === 'add') {
    inkLayer.appendChild(entry.element);
  } else {
    for (const el of entry.removed) {
      el.remove();
    }
  }
  undoStack.push(entry);
}

// ── Geometry helpers ───────────────────────────────────────────

function toSVGPoint(svg: SVGSVGElement, clientX: number, clientY: number): DOMPoint {
  const pt = svg.createSVGPoint();
  pt.x = clientX;
  pt.y = clientY;
  return pt.matrixTransform(svg.getScreenCTM()!.inverse());
}

function makePenPath(p: PenSettings, x: number, y: number): SVGPathElement {
  const el = document.createElementNS('http://www.w3.org/2000/svg', 'path');
  el.setAttribute('fill',            'none');
  el.setAttribute('stroke',          p.color);
  el.setAttribute('stroke-opacity',  String(p.opacity));
  el.setAttribute('stroke-width',    String(p.size));
  el.setAttribute('stroke-linecap',  'round');
  el.setAttribute('stroke-linejoin', 'round');
  el.setAttribute('d',               `M ${x} ${y}`);
  return el;
}

// ── Serialization ──────────────────────────────────────────────

// serializeDrawing returns the inner SVG markup to persist, or null when the
// ink layer is empty (nothing to save).
function serializeDrawing(inkLayer: SVGGElement): string | null {
  if (inkLayer.childElementCount === 0) return null;
  return inkLayer.outerHTML;
}

// restoreDrawing parses previously saved markup and populates the ink layer.
// Only called after validateSVG() has confirmed the markup is well-formed.
function restoreDrawing(data: string, inkLayer: SVGGElement): void {
  try {
    const parser = new DOMParser();
    const doc = parser.parseFromString(
      `<svg xmlns="http://www.w3.org/2000/svg">${data}</svg>`,
      'image/svg+xml',
    );

    const savedInk = doc.querySelector('#drawing-ink');
    if (savedInk) {
      savedInk.childNodes.forEach(node => {
        inkLayer.appendChild(document.importNode(node, true));
      });
    }
  } catch (err) {
    console.error('Failed to restore drawing:', err);
  }
}
