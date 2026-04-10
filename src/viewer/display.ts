// Image display: wheel zoom, drag to pan, double-click to reset.
// Zoom is anchored to the cursor position so the point under the cursor
// stays fixed as the scale changes.

interface ZoomState {
  scale: number;
  x: number;
  y: number;
}

const MIN_SCALE = 0.2;
const MAX_SCALE = 10;
const ZOOM_FACTOR = 0.1; // scale change per wheel tick (10%)

export function initImageDisplay() {
  const image = document.getElementById('page-image') as HTMLImageElement;

  if (!image) {
    return;
  }

  image.addEventListener('error', () => {
    console.error('Failed to load image');
  });

  const state: ZoomState = { scale: 1, x: 0, y: 0 };

  function applyTransform(): void {
    image.style.transform =
      `translate(${state.x}px, ${state.y}px) scale(${state.scale})`;
  }

  function resetTransform(): void {
    state.scale = 1;
    state.x = 0;
    state.y = 0;
    image.style.transform = '';
    image.style.cursor = '';
  }

  // Wheel zoom anchored to the cursor position.
  // Attached to the container so zoom still works when the drawing overlay
  // sits on top and intercepts pointer events on the image.
  const wheelTarget = (image.closest('.viewer-image') as HTMLElement) ?? image;
  wheelTarget.addEventListener('wheel', (e: WheelEvent) => {
    e.preventDefault();

    const prevScale = state.scale;
    const delta = e.deltaY > 0 ? 1 - ZOOM_FACTOR : 1 + ZOOM_FACTOR;
    state.scale = Math.min(Math.max(prevScale * delta, MIN_SCALE), MAX_SCALE);

    // Offset the translation so the point under the cursor stays fixed.
    const rect = image.getBoundingClientRect();
    const cursorX = e.clientX - rect.left - rect.width / 2;
    const cursorY = e.clientY - rect.top - rect.height / 2;
    const scaleDiff = state.scale / prevScale;
    state.x = cursorX + (state.x - cursorX) * scaleDiff;
    state.y = cursorY + (state.y - cursorY) * scaleDiff;

    applyTransform();
  }, { passive: false });

  // Drag to pan (left button only).
  let drag: { startX: number; startY: number; origX: number; origY: number } | null = null;

  image.addEventListener('mousedown', (e: MouseEvent) => {
    if (e.button !== 0) return;
    drag = { startX: e.clientX, startY: e.clientY, origX: state.x, origY: state.y };
    image.style.cursor = 'grabbing';
    e.preventDefault();
  });

  document.addEventListener('mousemove', (e: MouseEvent) => {
    if (!drag) return;
    state.x = drag.origX + (e.clientX - drag.startX);
    state.y = drag.origY + (e.clientY - drag.startY);
    applyTransform();
  });

  document.addEventListener('mouseup', () => {
    if (!drag) return;
    image.style.cursor = state.scale === 1 ? '' : 'grab';
    drag = null;
  });

  // Double-click resets zoom and position.
  image.addEventListener('dblclick', resetTransform);
}
