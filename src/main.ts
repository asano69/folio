import { initViewer } from './viewer/navigation';
import { initImageDisplay } from './viewer/display';
import { initRename } from './ui/rename';
import { initEditor } from './viewer/editor';
import { initTOC } from './viewer/toc';

document.addEventListener('DOMContentLoaded', () => {
    initViewer();
    initImageDisplay();
    initRename();
    initEditor();
    initTOC();
});
