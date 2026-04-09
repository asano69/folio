// Toggle handler for the TOC side pane in the viewer.
export function initTOC(): void {
    const toggleBtn = document.getElementById('toc-toggle');
    const closeBtn = document.getElementById('toc-close');
    const pane = document.getElementById('toc-pane');

    if (!pane) return;

    const open = () => {
        pane.hidden = false;
        toggleBtn?.classList.add('active');
    };

    const close = () => {
        pane.hidden = true;
        toggleBtn?.classList.remove('active');
    };

    toggleBtn?.addEventListener('click', open);
    closeBtn?.addEventListener('click', close);
}
