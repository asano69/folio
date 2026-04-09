import { initViewer } from './viewer/navigation';
import { initRename } from './ui/rename';
import { initEditor } from './viewer/editor';
import { initTOC } from './viewer/toc';

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    initViewer();
    initRename();
    initEditor();
    initTOC();
});
