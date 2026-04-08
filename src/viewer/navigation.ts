export function initViewer() {
  // ページセレクタの処理
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

  // キーボードショートカット
  document.addEventListener('keydown', handleKeyboardNavigation);
}

function handleKeyboardNavigation(e: KeyboardEvent) {
  // ビューアページでのみ動作
  if (!document.querySelector('.viewer-container')) {
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
