// Search and filter logic for the book library.
//
// FilterCriteria is the single source of truth for what the user is searching
// for. Add new fields here when introducing author, year, tag, etc. filtering.
export interface FilterCriteria {
  title: string;
  // Planned fields (Phase 2+):
  // author: string;
  // yearFrom: number | null;
  // yearTo:   number | null;
  // tags: string[];
}

function emptyCriteria(): FilterCriteria {
  return { title: '' };
}

function buildCriteria(): FilterCriteria {
  const titleInput = document.getElementById('search-title') as HTMLInputElement | null;
  return {
    title: titleInput?.value.trim().toLowerCase() ?? '',
  };
}

function isActive(criteria: FilterCriteria): boolean {
  return criteria.title !== '';
  // Extend this check as new fields are added.
}

function matchesCard(card: HTMLElement, criteria: FilterCriteria): boolean {
  if (criteria.title) {
    const titleEl = card.querySelector('.book-title');
    const title = (titleEl?.textContent ?? '').trim().toLowerCase();
    if (!title.includes(criteria.title)) return false;
  }
  // Add further field checks here as FilterCriteria grows.
  return true;
}

export function initSearch(): void {
  const titleInput = document.getElementById('search-title') as HTMLInputElement | null;
  if (!titleInput) return;

  const grid = document.querySelector<HTMLElement>('.books-grid:not(.missing-grid)');
  const noResultsMsg = document.getElementById('search-no-results');
  const clearBtn = document.getElementById('search-clear') as HTMLButtonElement | null;

  const applyFilter = (): void => {
    const criteria = buildCriteria();

    // Toggle the clear button visibility.
    if (clearBtn) clearBtn.hidden = !isActive(criteria);

    if (!grid) return;

    const cards = grid.querySelectorAll<HTMLElement>('.book-card');
    let visibleCount = 0;

    cards.forEach(card => {
      const visible = matchesCard(card, criteria);
      card.hidden = !visible;
      if (visible) visibleCount++;
    });

    if (noResultsMsg) noResultsMsg.hidden = visibleCount > 0;
  };

  titleInput.addEventListener('input', applyFilter);

  clearBtn?.addEventListener('click', () => {
    titleInput.value = '';
    applyFilter();
    titleInput.focus();
  });

  // Pressing Escape clears the search input.
  titleInput.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      titleInput.value = '';
      applyFilter();
    }
  });
}
