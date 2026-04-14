// src/viewer/navigation.ts

// Page selector handler
export function initViewer() {
  const pageSelect = document.getElementById('page-select') as HTMLSelectElement;

  if (pageSelect) {
    pageSelect.addEventListener('change', (e) => {
      const target = e.target as HTMLSelectElement;
      const bookId = target.dataset.bookId;
      const seq = target.value;

      if (bookId && seq) {
        window.location.href = `/books/${bookId}?seq=${seq}`;
      }
    });
  }

  // Keyboard navigation is handled centrally by the keyboard manager
  // in src/keyboard/init.ts, so we don't need to duplicate logic here.
}
