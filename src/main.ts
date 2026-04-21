import { initKeyboard } from './keyboard/init';
import { initViewer } from './viewer/navigation';
import { initImageDisplay } from './viewer/display';
import { initRename } from './ui/rename';
import { initEditor } from './viewer/editor';
import { initTOC } from './viewer/toc';
import { initSearch } from './ui/search';
import { initCollections } from './ui/collections';
import { initDrawing } from './viewer/drawing';
import { initPageStatus } from './ui/page-status';
import { initBookMeta } from './ui/book-meta';
import { initLibrary } from './library';

document.addEventListener('DOMContentLoaded', () => {
    // Initialize keyboard bindings first, so all other modules can assume it's ready.
    initKeyboard();

    initViewer();
    initImageDisplay();
    initRename();
    initEditor();
    initTOC();
    initSearch();
    initCollections();
    initDrawing();
    initPageStatus();
    initBookMeta();
    initLibrary();
});
