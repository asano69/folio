// Toggle handler for the collapsible TOC overlay in the viewer.
export function initTOC(): void {
    const toggleBtn = document.getElementById('toc-toggle');
    const closeBtn  = document.getElementById('toc-close');
    const pane      = document.getElementById('toc-pane');
    const backdrop  = document.getElementById('toc-backdrop');

    if (!pane) return;

    const open = () => {
        pane.classList.add('open');
        backdrop?.classList.add('visible');
        toggleBtn?.classList.add('active');
    };

    const close = () => {
        pane.classList.remove('open');
        backdrop?.classList.remove('visible');
        toggleBtn?.classList.remove('active');
    };

    toggleBtn?.addEventListener('click', () => {
        pane.classList.contains('open') ? close() : open();
    });
    closeBtn?.addEventListener('click', close);
    // Clicking the backdrop closes the pane.
    backdrop?.addEventListener('click', close);
}
