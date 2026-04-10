// SVG drawing overlay for the page viewer.
//
// Architecture:
//   - A transparent SVG element is absolutely positioned over the page image.
//   - The SVG shares the image's CSS transform (synced via MutationObserver) so
//     that zoom and pan apply to both simultaneously.
//   - Ink strokes live in <g id="drawing-ink">; eraser strokes live in
//     <g id="drawing-erase" style="mix-blend-mode: destination-out">.
//   - The SVG has isolation: isolate so destination-out only erases within the
//     SVG's compositing context, never the underlying page image.
//
// Known limitation: because ink and erase strokes are stored in separate layers,
// eraser strokes always apply to all ink regardless of draw order. Drawing ink
// over an erased area then undoing the erase will reveal the ink underneath.
// For typical annotation use this trade-off is acceptable.

import { saveDrawing } from '../api';

type LayerName = 'ink' | 'erase';

interface Stroke {
  element: SVGPathElement;
  layer:   LayerName;
}

interface PenSettings {
  color:   string;
  opacity: number; // 0–1
  size:    number;
}

interface EraserSettings {
  size: number;
}

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

  const inkLayer   = overlay.querySelector<SVGGElement>('#drawing-ink')!;
  const eraseLayer = overlay.querySelector<SVGGElement>('#drawing-erase')!;

  // ── State ──────────────────────────────────────────────────────────────────

  const pen:    PenSettings    = { color: '#e74c3c', opacity: 1, size: 4 };
  const eraser: EraserSettings = { size: 20 };
  let activeTool: LayerName    = 'ink';

  const undoStack: Stroke[] = [];
  const redoStack: Stroke[] = [];

  // ── SVG viewBox ────────────────────────────────────────────────────────────

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
    restoreDrawing(existingData, inkLayer, eraseLayer);
  }

  // ── Pane open / close ──────────────────────────────────────────────────────

  const setDrawingMode = (active: boolean): void => {
    overlay.style.pointerEvents = active ? 'all'        : 'none';
    overlay.style.cursor        = active ? 'crosshair'  : '';
    overlay.style.touchAction   = active ? 'none'       : '';
  };

  const openPane = (): void => {
    // Close the edit pane if it is open, without triggering its snapshot logic.
    document.dispatchEvent(new CustomEvent('folio:draw-pane-open'));
    pane.classList.add('open');
    backdrop?.classList.add('visible');
    toggleBtn.classList.add('active');
    setDrawingMode(true);
  };

  const closePane = (): void => {
    pane.classList.remove('open');
    backdrop?.classList.remove('visible');
    toggleBtn.classList.remove('active');
    setDrawingMode(false);
  };

  toggleBtn.addEventListener('click', () => {
    pane.classList.contains('open') ? closePane() : openPane();
  });
  closeBtn?.addEventListener('click',  closePane);
  backdrop?.addEventListener('click',  closePane);

  // Close when the edit pane is opened.
  document.addEventListener('folio:edit-pane-open', closePane);

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

  penBtn?.addEventListener('click', () => { activeTool = 'ink';   syncToolUI(); });
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

  // ── Save ───────────────────────────────────────────────────────────────────

  saveBtn?.addEventListener('click', async () => {
    if (!saveBtn) return;
    saveBtn.disabled = true;
    try {
      const svg = serializeDrawing(inkLayer, eraseLayer);
      await saveDrawing(bookId, pageHash, svg);
    } catch (err) {
      console.error('Failed to save drawing:', err);
    } finally {
      saveBtn.disabled = false;
    }
  });

  // ── Undo / redo ────────────────────────────────────────────────────────────

  document.addEventListener('keydown', (e: KeyboardEvent) => {
    if (!pane.classList.contains('open')) return;
    const active = document.activeElement;
    if (active && (active.tagName === 'INPUT' || active.tagName === 'TEXTAREA')) return;

    if (e.ctrlKey && !e.shiftKey && e.key === 'z') {
      e.preventDefault();
      undoStroke(undoStack, redoStack);
    } else if (e.ctrlKey && (e.key === 'y' || (e.shiftKey && e.key === 'Z'))) {
      e.preventDefault();
      redoStroke(undoStack, redoStack, inkLayer, eraseLayer);
    }
  });

  // ── Drawing interaction ────────────────────────────────────────────────────

  let currentPath:  SVGPathElement | null = null;
  let currentLayer: SVGGElement    | null = null;
  let pathData = '';
  let drawing  = false;

  overlay.addEventListener('pointerdown', (e: PointerEvent) => {
    if (e.button !== 0) return;
    e.preventDefault();
    overlay.setPointerCapture(e.pointerId);

    // Any new stroke invalidates the redo history.
    redoStack.splice(0);

    const pt = toSVGPoint(overlay, e.clientX, e.clientY);

    if (activeTool === 'ink') {
      currentPath  = makePenPath(pen, pt.x, pt.y);
      currentLayer = inkLayer;
    } else {
      currentPath  = makeEraserPath(eraser, pt.x, pt.y);
      currentLayer = eraseLayer;
    }

    pathData = `M ${pt.x} ${pt.y}`;
    currentPath.setAttribute('d', pathData);
    currentLayer.appendChild(currentPath);
    drawing = true;
  });

  overlay.addEventListener('pointermove', (e: PointerEvent) => {
    if (!drawing || !currentPath) return;
    const pt  = toSVGPoint(overlay, e.clientX, e.clientY);
    pathData += ` L ${pt.x} ${pt.y}`;
    currentPath.setAttribute('d', pathData);
  });

  const endStroke = (): void => {
    if (!drawing || !currentPath || !currentLayer) return;
    drawing = false;
    undoStack.push({ element: currentPath, layer: activeTool });
    currentPath  = null;
    currentLayer = null;
    pathData     = '';
  };

  overlay.addEventListener('pointerup',     endStroke);
  overlay.addEventListener('pointercancel', endStroke);
}

// ── Helpers ────────────────────────────────────────────────────────────────────

function toSVGPoint(svg: SVGSVGElement, clientX: number, clientY: number): DOMPoint {
  const pt = svg.createSVGPoint();
  pt.x = clientX;
  pt.y = clientY;
  return pt.matrixTransform(svg.getScreenCTM()!.inverse());
}

function makePenPath(p: PenSettings, x: number, y: number): SVGPathElement {
  const el = document.createElementNS('http://www.w3.org/2000/svg', 'path');
  el.setAttribute('fill',              'none');
  el.setAttribute('stroke',            p.color);
  el.setAttribute('stroke-opacity',    String(p.opacity));
  el.setAttribute('stroke-width',      String(p.size));
  el.setAttribute('stroke-linecap',    'round');
  el.setAttribute('stroke-linejoin',   'round');
  el.setAttribute('d',                 `M ${x} ${y}`);
  return el;
}

function makeEraserPath(e: EraserSettings, x: number, y: number): SVGPathElement {
  const el = document.createElementNS('http://www.w3.org/2000/svg', 'path');
  el.setAttribute('fill',            'none');
  el.setAttribute('stroke',          'black'); // color is irrelevant; destination-out uses alpha
  el.setAttribute('stroke-width',    String(e.size));
  el.setAttribute('stroke-linecap',  'round');
  el.setAttribute('stroke-linejoin', 'round');
  el.setAttribute('d',               `M ${x} ${y}`);
  return el;
}

function undoStroke(undoStack: Stroke[], redoStack: Stroke[]): void {
  const stroke = undoStack.pop();
  if (!stroke) return;
  stroke.element.remove();
  redoStack.push(stroke);
}

function redoStroke(
  undoStack: Stroke[],
  redoStack: Stroke[],
  inkLayer:   SVGGElement,
  eraseLayer: SVGGElement,
): void {
  const stroke = redoStack.pop();
  if (!stroke) return;
  const layer = stroke.layer === 'ink' ? inkLayer : eraseLayer;
  layer.appendChild(stroke.element);
  undoStack.push(stroke);
}

// serializeDrawing returns the inner SVG markup to persist, or null when the
// ink layer is empty (nothing to save).
function serializeDrawing(inkLayer: SVGGElement, eraseLayer: SVGGElement): string | null {
  if (inkLayer.childElementCount === 0) return null;
  return inkLayer.outerHTML + eraseLayer.outerHTML;
}

// restoreDrawing parses previously saved markup and populates the live layers.
// Restored strokes are intentionally excluded from the undo stack so that
// Ctrl+Z only operates on strokes drawn in the current session.
function restoreDrawing(
  data:       string,
  inkLayer:   SVGGElement,
  eraseLayer: SVGGElement,
): void {
  const parser = new DOMParser();
  const doc    = parser.parseFromString(
    `<svg xmlns="http://www.w3.org/2000/svg">${data}</svg>`,
    'image/svg+xml',
  );

  const savedInk   = doc.querySelector('#drawing-ink');
  const savedErase = doc.querySelector('#drawing-erase');

  savedInk?.childNodes.forEach(node => {
    inkLayer.appendChild(document.importNode(node, true));
  });
  savedErase?.childNodes.forEach(node => {
    eraseLayer.appendChild(document.importNode(node, true));
  });
}
