import { useCallback, useEffect, useRef } from 'react';
import type { Diagram } from '../../api/types';
import { copySelection, nudgeNodes, pasteClip, removeNodes, PASTE_OFFSET, type Clip } from '../../equipment/clipboard';

type Options = {
  active: boolean; locked: boolean; diagram: Diagram; selection: string[];
  edit: (diagram: Diagram, group?: string) => void; setSelection: (ids: string[]) => void;
  undo: () => void; redo: () => void;
};
const ARROWS: Record<string, [number, number]> = {
  ArrowLeft: [-1, 0], ArrowRight: [1, 0], ArrowUp: [0, -1], ArrowDown: [0, 1],
};
const GRID = 20;

function typing(event: KeyboardEvent): boolean {
  const target = event.target as HTMLElement;
  return !!document.querySelector('dialog[open]') || target.isContentEditable ||
    ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) || !!window.getSelection()?.toString();
}

/** Canvas editing actions shared by keyboard shortcuts and the properties panel. */
export function useCanvasShortcuts({ active, locked, diagram, selection, edit, setSelection, undo, redo }: Options) {
  const clip = useRef<{ content: Clip; pastes: number } | null>(null);
  const duplicate = useCallback(() => {
    const content = copySelection(diagram, new Set(selection));
    if (locked || !content) return;
    const result = pasteClip(diagram, content, PASTE_OFFSET);
    edit(result.diagram); setSelection(result.ids);
  }, [diagram, edit, locked, selection, setSelection]);
  const remove = useCallback(() => {
    if (locked || !selection.length) return;
    edit(removeNodes(diagram, new Set(selection))); setSelection([]);
  }, [diagram, edit, locked, selection, setSelection]);
  const copy = useCallback(() => {
    const content = copySelection(diagram, new Set(selection));
    if (content) clip.current = { content, pastes: 0 };
    return !!content;
  }, [diagram, selection]);
  const paste = useCallback(() => {
    if (locked || !clip.current) return;
    clip.current.pastes += 1;
    const result = pasteClip(diagram, clip.current.content, PASTE_OFFSET * clip.current.pastes);
    edit(result.diagram); setSelection(result.ids);
  }, [diagram, edit, locked, setSelection]);

  useEffect(() => {
    if (!active) return;
    const keyboard = (event: KeyboardEvent) => {
      if (typing(event)) return;
      const key = event.key.toLowerCase();
      const command = event.ctrlKey || event.metaKey;
      if (event.key === 'Escape' && selection.length) { setSelection([]); return; }
      if (command && key === 'a') { event.preventDefault(); setSelection(diagram.nodes.map(node => node.id)); return; }
      if (command && key === 'c') { if (copy()) event.preventDefault(); return; }
      if (locked) return;
      if (command && key === 'z') { event.preventDefault(); if (event.shiftKey) redo(); else undo(); }
      else if (command && key === 'y') { event.preventDefault(); redo(); }
      else if (command && key === 'x') { if (copy()) { event.preventDefault(); remove(); } }
      else if (command && key === 'v') { event.preventDefault(); paste(); }
      else if (command && key === 'd') { event.preventDefault(); duplicate(); }
      else if (ARROWS[event.key] && selection.length && !command &&
          !(event.target as HTMLElement).closest('.react-flow__node, .react-flow__nodesselection')) {
        // React Flow already moves a focused node or selection box; only handle unfocused selections.
        event.preventDefault();
        const [dx, dy] = ARROWS[event.key];
        const step = event.shiftKey ? GRID * 5 : GRID;
        edit(nudgeNodes(diagram, new Set(selection), dx * step, dy * step), `nudge:${selection.join(',')}`);
      }
    };
    window.addEventListener('keydown', keyboard);
    return () => window.removeEventListener('keydown', keyboard);
  }, [active, copy, diagram, duplicate, edit, locked, paste, redo, remove, selection, setSelection, undo]);
  return { duplicate, remove };
}
