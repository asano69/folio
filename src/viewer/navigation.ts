export function initViewer() {
  // Page selector handler
  const pageSelect = document.getElementById('page-select') as HTMLSelectElement;

  if (pageSelect) {
    pageSelect.addEventListener('change', (e) => {
      const target = e.target as HTMLSelectElement;
      const bookId = target.dataset.bookId;
      const pageNum = target.value;

      if (bookId && pageNum) {
        window.location.href = `/viewer?book=${bookId}&page=${pageNum}`;
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
  }
}
