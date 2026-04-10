export function initViewer() {
  // Page selector handler
  const pageSelect = document.getElementById('page-select') as HTMLSelectElement;

  if (pageSelect) {
    pageSelect.addEventListener('change', (e) => {
      const target = e.target as HTMLSelectElement;
      const bookId = target.dataset.bookId;
      const pageNum = target.value;

      if (bookId && pageNum) {
        window.location.href = `/books/${bookId}/pages/${pageNum}`;
      }
    });
  }

  // Keyboard shortcuts
  document.addEventListener('keydown', handleKeyboardNavigation);
}

function handleKeyboardNavigation(e: KeyboardEvent) {
  // Only active on viewer pages
  if (!document.querySelector('.viewer-container')) {
    return;
  }

  // Ctrl+H: return to library
  if (e.ctrlKey && e.key === 'h') {
    e.preventDefault();
    window.location.href = '/';
    return;
  }

  // Ctrl+J: toggle the page jump overlay
  if (e.ctrlKey && e.key === 'j') {
    e.preventDefault();
    const controls = document.querySelector('.viewer-controls') as HTMLElement | null;
    if (controls) {
      const opening = !controls.classList.contains('visible');
      controls.classList.toggle('visible');
      if (opening) {
        // Focus the select immediately so the user can type/arrow without extra clicks
        const sel = controls.querySelector('select') as HTMLSelectElement | null;
        sel?.focus();
      }
    }
    return;
  }

  // Suppress page navigation while the drawing pane is active so that key
  // events intended for drawing (Ctrl+Z etc.) are not misinterpreted, and so
  // the user does not accidentally navigate away with unsaved strokes.
  if (document.getElementById('draw-pane')?.classList.contains('open')) {
    return;
  }

  // Suppress page navigation when focus is inside a form element
  const active = document.activeElement;
  if (active && (
    active.tagName === 'INPUT' ||
    active.tagName === 'TEXTAREA' ||
    active.tagName === 'SELECT'
  )) {
    return;
  }

  const prevBtn = document.querySelector('.prev-btn:not(.disabled)') as HTMLAnchorElement;
  const nextBtn = document.querySelector('.next-btn:not(.disabled)') as HTMLAnchorElement;

  switch (e.key) {
    case 'ArrowLeft':
    case 'h':
      if (prevBtn) {
        e.preventDefault();
        window.location.href = prevBtn.href;
      }
      break;

    case 'ArrowRight':
    case 'l':
      if (nextBtn) {
        e.preventDefault();
        window.location.href = nextBtn.href;
      }
      break;

    case 'Escape':
      // Close the jump overlay if it is open
      document.querySelector('.viewer-controls')?.classList.remove('visible');
      break;
  }
}
