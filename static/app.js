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

// src/ui/rename.ts
function initRename() {
  document.querySelectorAll(".rename-btn").forEach((btn) => {
    btn.addEventListener("click", (e) => {
      e.preventDefault();
      const bookId = btn.dataset.bookId;
      if (!bookId) return;
      const titleEl = document.querySelector(
        `.book-title[data-book-id="${bookId}"]`
      );
      if (!titleEl) return;
      startRename(titleEl, bookId);
    });
  });
}
async function startRename(titleEl, bookId) {
  const currentTitle = titleEl.textContent ?? "";
  const input = document.createElement("input");
  input.type = "text";
  input.value = currentTitle;
  input.className = "rename-input";
  titleEl.replaceWith(input);
  input.focus();
  input.select();
  let cancelled = false;
  let finishing = false;
  const finish = async () => {
    if (finishing) return;
    finishing = true;
    const newTitle = input.value.trim();
    if (!cancelled && newTitle && newTitle !== currentTitle) {
      try {
        const res = await fetch(`/api/books/${bookId}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ title: newTitle })
        });
        if (!res.ok) throw new Error(`rename failed: ${res.status}`);
        titleEl.textContent = newTitle;
      } catch (err) {
        console.error(err);
        titleEl.textContent = currentTitle;
      }
    } else {
      titleEl.textContent = currentTitle;
    }
    input.replaceWith(titleEl);
  };
  input.addEventListener("blur", finish);
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      input.blur();
    } else if (e.key === "Escape") {
      cancelled = true;
      input.blur();
    }
  });
}

// src/main.ts
document.addEventListener("DOMContentLoaded", () => {
  initViewer();
  initRename();
});
//# sourceMappingURL=app.js.map
