// Toggle handler for the collapsible TOC sidebar in the viewer.
export function initTOC(): void {
    const toggleBtn = document.getElementById('toc-toggle');
    const closeBtn = document.getElementById('toc-close');
    const pane = document.getElementById('toc-pane');

    if (!pane) return;

    const open = () => {
        pane.classList.add('open');
        toggleBtn?.classList.add('active');
    };

    const close = () => {
        pane.classList.remove('open');
        toggleBtn?.classList.remove('active');
    };

    toggleBtn?.addEventListener('click', () => {
        pane.classList.contains('open') ? close() : open();
    });
    closeBtn?.addEventListener('click', close);
}
