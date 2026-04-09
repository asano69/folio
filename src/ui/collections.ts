// Collections sidebar: drag-and-drop, create, rename, and delete.

export function initCollections(): void {
		setupSidebarToggle();
    setupDragAndDrop();
    setupCreateCollection();
    setupCollectionActions();
    setupRemoveFromCollection();
}

// ── Drag and drop ────────────────────────────────────────────

function setupDragAndDrop(): void {
    // Book cards are drag sources.
    document.querySelectorAll<HTMLElement>('.book-card[data-book-id]').forEach(card => {
        card.addEventListener('dragstart', (e: DragEvent) => {
            e.dataTransfer!.setData('text/plain', card.dataset.bookId!);
            e.dataTransfer!.effectAllowed = 'copy';
            card.classList.add('dragging');
        });
        card.addEventListener('dragend', () => card.classList.remove('dragging'));
    });

    // Collection items are drop targets.
    document.querySelectorAll<HTMLElement>('.collection-drop-zone').forEach(zone => {
        zone.addEventListener('dragover', (e: DragEvent) => {
            e.preventDefault();
            e.dataTransfer!.dropEffect = 'copy';
            zone.classList.add('drag-over');
        });
        zone.addEventListener('dragleave', () => zone.classList.remove('drag-over'));
        zone.addEventListener('drop', (e: DragEvent) => {
            e.preventDefault();
            zone.classList.remove('drag-over');
            const bookId = e.dataTransfer!.getData('text/plain');
            const collectionId = zone.dataset.collectionId!;
            if (bookId && collectionId) {
                addBookToCollection(zone, collectionId, bookId);
            }
        });
    });
}

async function addBookToCollection(
    zone: HTMLElement,
    collectionId: string,
    bookId: string,
): Promise<void> {
    try {
        const res = await fetch(`/api/collections/${collectionId}/books/${bookId}`, {
            method: 'POST',
        });
        if (!res.ok) throw new Error(`add failed: ${res.status}`);

        // Increment the displayed book count without a full reload.
        const countEl = zone.querySelector<HTMLElement>('.collection-count');
        if (countEl) {
            const n = parseInt(countEl.textContent?.match(/\d+/)?.[0] ?? '0', 10);
            countEl.textContent = `(${n + 1})`;
        }

        zone.classList.add('drop-success');
        setTimeout(() => zone.classList.remove('drop-success'), 700);
    } catch (err) {
        console.error(err);
    }
}

// ── Create collection ─────────────────────────────────────────

function setupCreateCollection(): void {
    const btn = document.getElementById('collection-new-btn');
    if (!btn) return;

    btn.addEventListener('click', () => {
        const input = document.createElement('input');
        input.type = 'text';
        input.className = 'collection-new-input';
        input.placeholder = 'Collection name';

        btn.replaceWith(input);
        input.focus();

        let finishing = false;

        const finish = async (): Promise<void> => {
            if (finishing) return;
            finishing = true;

            const title = input.value.trim();
            if (title) {
                try {
                    const res = await fetch('/api/collections/', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ title }),
                    });
                    if (!res.ok) throw new Error(`create failed: ${res.status}`);
                    window.location.reload();
                    return;
                } catch (err) {
                    console.error(err);
                }
            }
            input.replaceWith(btn);
        };

        input.addEventListener('blur', finish);
        input.addEventListener('keydown', (e: KeyboardEvent) => {
            if (e.key === 'Enter') { e.preventDefault(); input.blur(); }
            if (e.key === 'Escape') { input.value = ''; input.blur(); }
        });
    });
}

// ── Rename and delete ─────────────────────────────────────────

function setupCollectionActions(): void {
    document.querySelectorAll<HTMLButtonElement>('.collection-rename-btn').forEach(btn => {
        btn.addEventListener('click', (e: Event) => {
            e.preventDefault();
            e.stopPropagation();
            const item = btn.closest<HTMLElement>('.collection-drop-zone');
            const titleEl = item?.querySelector<HTMLElement>('.collection-title');
            if (!item || !titleEl) return;
            startRenameCollection(item.dataset.collectionId!, titleEl);
        });
    });

    document.querySelectorAll<HTMLButtonElement>('.collection-delete-btn').forEach(btn => {
        btn.addEventListener('click', async (e: Event) => {
            e.preventDefault();
            e.stopPropagation();
            const item = btn.closest<HTMLElement>('.collection-drop-zone');
            if (!item) return;
            const collectionId = item.dataset.collectionId!;

            try {
                const res = await fetch(`/api/collections/${collectionId}`, {
                    method: 'DELETE',
                });
                if (!res.ok) throw new Error(`delete failed: ${res.status}`);

                // If the deleted collection is currently active, return to All Books.
                const params = new URLSearchParams(window.location.search);
                if (params.get('collection') === collectionId) {
                    window.location.href = '/';
                } else {
                    item.remove();
                }
            } catch (err) {
                console.error(err);
            }
        });
    });
}

async function startRenameCollection(
    collectionId: string,
    titleEl: HTMLElement,
): Promise<void> {
    const currentTitle = titleEl.textContent ?? '';

    const input = document.createElement('input');
    input.type = 'text';
    input.value = currentTitle;
    input.className = 'collection-rename-input';

    titleEl.replaceWith(input);
    input.focus();
    input.select();

    let cancelled = false;
    let finishing = false;

    const finish = async (): Promise<void> => {
        if (finishing) return;
        finishing = true;

        const newTitle = input.value.trim();
        if (!cancelled && newTitle && newTitle !== currentTitle) {
            try {
                const res = await fetch(`/api/collections/${collectionId}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ title: newTitle }),
                });
                if (!res.ok) throw new Error(`rename failed: ${res.status}`);
                titleEl.textContent = newTitle;
            } catch (err) {
                console.error(err);
            }
        }
        input.replaceWith(titleEl);
    };

    input.addEventListener('blur', finish);
    input.addEventListener('keydown', (e: KeyboardEvent) => {
        if (e.key === 'Enter') { e.preventDefault(); input.blur(); }
        if (e.key === 'Escape') { cancelled = true; input.blur(); }
    });
}

// ── Remove book from collection ───────────────────────────────

function setupRemoveFromCollection(): void {
    document.querySelectorAll<HTMLButtonElement>('.collection-remove-btn').forEach(btn => {
        btn.addEventListener('click', async (e: Event) => {
            e.preventDefault();
            e.stopPropagation();
            const { bookId, collectionId } = btn.dataset;
            if (!bookId || !collectionId) return;

            try {
                const res = await fetch(`/api/collections/${collectionId}/books/${bookId}`, {
                    method: 'DELETE',
                });
                if (!res.ok) throw new Error(`remove failed: ${res.status}`);
                btn.closest<HTMLElement>('.book-card')?.remove();
            } catch (err) {
                console.error(err);
            }
        });
    });
}

// ── Sidebar toggle ────────────────────────────────────────────

function setupSidebarToggle(): void {
    const sidebar = document.getElementById('collection-sidebar');
    const btn = document.getElementById('sidebar-toggle');
    if (!sidebar || !btn) return;

    const STORAGE_KEY = 'folio:sidebar-collapsed';
    const collapsed = localStorage.getItem(STORAGE_KEY) === 'true';
    if (collapsed) {
        sidebar.classList.add('collection-sidebar--collapsed');
    }

    btn.addEventListener('click', () => {
        const isCollapsed = sidebar.classList.toggle('collection-sidebar--collapsed');
        localStorage.setItem(STORAGE_KEY, String(isCollapsed));
    });
}
