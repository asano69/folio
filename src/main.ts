import { initViewer } from './viewer/navigation';
import { initImageDisplay } from './viewer/display';
import { initRename } from './ui/rename';
import { initEditor } from './viewer/editor';
import { initTOC } from './viewer/toc';
import { initSearch } from './ui/search';
import { initCollections } from './ui/collections';
import { initDrawing } from './viewer/drawing';
import { initPageStatus } from './ui/page-status';
import { initBookNote } from './ui/book-note';

document.addEventListener('DOMContentLoaded', () => {
    initViewer();
    initImageDisplay();
    initRename();
    initEditor();
    initTOC();
    initSearch();
    initCollections();
    initDrawing();
    initPageStatus();
    initBookNote();
});
