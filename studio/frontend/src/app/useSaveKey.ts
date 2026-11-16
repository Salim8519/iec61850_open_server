import { useEffect } from 'react';

/** Ctrl+S saves the project (ADR-0139), like File › Save project; it never opens the browser's
 *  own "save page" box. Inside a dialog it does nothing. */
export function useSaveKey(save: () => void, blocked: boolean) {
  useEffect(() => {
    const keyboard = (event: KeyboardEvent) => {
      if (!(event.ctrlKey || event.metaKey) || event.altKey || event.shiftKey || event.key.toLowerCase() !== 's') return;
      event.preventDefault();
      if (!blocked && !document.querySelector('dialog[open]')) save();
    };
    window.addEventListener('keydown', keyboard);
    return () => window.removeEventListener('keydown', keyboard);
  }, [save, blocked]);
}
