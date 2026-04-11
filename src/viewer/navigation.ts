// src/viewer/navigation.ts

// Page selector handler
export function initViewer() {
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

  // Keyboard navigation is now handled centrally by the keyboard manager
  // in src/keyboard/init.ts, so we don't need to duplicate logic here.
}
