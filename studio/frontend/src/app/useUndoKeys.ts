import { useEffect } from 'react';
import type { ProjectHistory } from '../state/useHistory';

/** True while the user types or a dialog is open: the browser's own text Undo applies then. */
function typing(event: KeyboardEvent): boolean {
  const target = event.target as HTMLElement;
  return !!document.querySelector('dialog[open]') || target.isContentEditable ||
    ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName);
}

/** Ctrl+Z undoes and Ctrl+Y or Ctrl+Shift+Z redoes the last project change on every page
 *  (ADR-0135); the engine says what can be undone. */
export function useUndoKeys(history: ProjectHistory, busy: boolean) {
  useEffect(() => {
    const keyboard = (event: KeyboardEvent) => {
      if (!(event.ctrlKey || event.metaKey) || event.altKey || typing(event)) return;
      const key = event.key.toLowerCase();
      const redo = key === 'y' || (key === 'z' && event.shiftKey);
      if (key !== 'z' && key !== 'y') return;
      event.preventDefault();
      if (busy) return;
      if (redo) { if (history.canRedo) history.redo(); } else if (history.canUndo) history.undo();
    };
    window.addEventListener('keydown', keyboard);
    return () => window.removeEventListener('keydown', keyboard);
  }, [history, busy]);
}
