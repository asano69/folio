// src/viewer/navigation.ts
function initViewer() {
  const pageSelect = document.getElementById("page-select");
  if (pageSelect) {
    pageSelect.addEventListener("change", (e) => {
      const target = e.target;
      const bookId = target.dataset.bookId;
      const pageNum = target.value;
      if (bookId && pageNum) {
        window.location.href = `/viewer?book=${bookId}&page=${pageNum}`;
      }
    });
  }
  document.addEventListener("keydown", handleKeyboardNavigation);
}
function handleKeyboardNavigation(e) {
  if (!document.querySelector(".viewer-container")) {
    return;
  }
  const prevBtn = document.querySelector(".prev-btn:not(.disabled)");
  const nextBtn = document.querySelector(".next-btn:not(.disabled)");
  switch (e.key) {
    case "ArrowLeft":
    case "h":
      if (prevBtn) {
        e.preventDefault();
        window.location.href = prevBtn.href;
      }
      break;
    case "ArrowRight":
    case "l":
      if (nextBtn) {
        e.preventDefault();
        window.location.href = nextBtn.href;
      }
      break;
  }
}

// src/main.ts
document.addEventListener("DOMContentLoaded", () => {
  initViewer();
});
//# sourceMappingURL=app.js.map
