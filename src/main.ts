import { initViewer } from './viewer/navigation';
import { initImageDisplay } from './viewer/display';
import { initRename } from './ui/rename';
import { initEditor } from './viewer/editor';
import { initTOC } from './viewer/toc';
import { initSearch } from './ui/search';
import { initCollections } from './ui/collections';
import { initDrawing } from './viewer/drawing';

document.addEventListener('DOMContentLoaded', () => {
    initViewer();
    initImageDisplay();
    initRename();
    initEditor();
    initTOC();
    initSearch();
    initCollections();
    initDrawing();
});
