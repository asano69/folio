import { initViewer } from './viewer/navigation';
import { initRename } from './ui/rename';

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
  initViewer();
  initRename();
});
