export function initEditor(): void {
    const toggleBtn = document.getElementById('edit-toggle') as HTMLButtonElement | null;
    const panel = document.getElementById('edit-panel') as HTMLElement | null;

    if (!toggleBtn || !panel) return;

    const bookId = panel.dataset.bookId!;
    const pageHash = panel.dataset.pageHash!;

    const titleInput = document.getElementById('edit-title') as HTMLInputElement;
    const attributeSelect = document.getElementById('edit-attribute') as HTMLSelectElement;
    const bodyTextarea = document.getElementById('edit-body') as HTMLTextAreaElement;
    const saveBtn = document.getElementById('edit-save') as HTMLButtonElement;
    const cancelBtn = document.getElementById('edit-cancel') as HTMLButtonElement;

    // Snapshot of field values at the moment the panel was opened.
    let snapshot = captureValues();

    toggleBtn.addEventListener('click', () => {
        if (panel.hidden) {
            openPanel();
        } else {
            restoreSnapshot();
            closePanel();
        }
    });

    saveBtn.addEventListener('click', () => { save(); });

    cancelBtn.addEventListener('click', () => {
        restoreSnapshot();
        closePanel();
    });

    function captureValues() {
        return {
            title: titleInput.value,
            attribute: attributeSelect.value,
            body: bodyTextarea.value,
        };
    }

    function restoreSnapshot(): void {
        titleInput.value = snapshot.title;
        attributeSelect.value = snapshot.attribute;
        bodyTextarea.value = snapshot.body;
    }

    function openPanel(): void {
        snapshot = captureValues();
        panel.hidden = false;
        toggleBtn.classList.add('active');
        titleInput.focus();
    }

    function closePanel(): void {
        panel.hidden = true;
        toggleBtn.classList.remove('active');
    }

    async function save(): Promise<void> {
        saveBtn.disabled = true;
        try {
            const res = await fetch(`/api/pages/${bookId}/${pageHash}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    title: titleInput.value.trim(),
                    attribute: attributeSelect.value,
                    body: bodyTextarea.value,
                }),
            });
            if (!res.ok) throw new Error(`save failed: ${res.status}`);

            snapshot = captureValues();
            updateDisplay(snapshot.title, snapshot.attribute, snapshot.body);
            closePanel();
        } catch (err) {
            console.error(err);
        } finally {
            saveBtn.disabled = false;
        }
    }

    function updateDisplay(title: string, attribute: string, body: string): void {
        const titleBadge = document.getElementById('page-title-badge') as HTMLElement | null;
        const attrBadge = document.getElementById('page-attr-badge') as HTMLElement | null;
        const noteEl = document.getElementById('page-note') as HTMLElement | null;
        const noteBody = document.getElementById('note-body') as HTMLElement | null;

        if (titleBadge) {
            titleBadge.textContent = title;
            titleBadge.hidden = !title;
        }

        if (attrBadge) {
            // Remove any previous attr-* class before applying the new one.
            attrBadge.className = 'page-attr-badge';
            if (attribute) {
                attrBadge.classList.add(`attr-${attribute}`);
                attrBadge.textContent = attribute;
                attrBadge.hidden = false;
            } else {
                attrBadge.hidden = true;
            }
        }

        if (noteEl && noteBody) {
            noteBody.textContent = body;
            noteEl.hidden = !body;
        }
    }
}
