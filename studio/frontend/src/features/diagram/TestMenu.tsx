import { useEffect, useRef } from 'react';
import type { EquipmentNode, Project } from '../../api/types';
import { TEST_COPY, testAvailability } from '../../equipment/testButtons';
import type { RunTest } from './EquipmentTest';

export type MenuAt = { nodeId: string; x: number; y: number };

/** Right-click menu on an equipment while running: the same test buttons as the side sheet.
 * "Set value" needs a number, so it opens the side sheet instead of guessing a value. */
export function TestMenu({ at, node, project, running, test, openSheet, close }: {
  at: MenuAt; node: EquipmentNode; project: Project; running: Set<string>; test: RunTest;
  openSheet: () => void; close: () => void;
}) {
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const done = (event: MouseEvent | KeyboardEvent) => {
      if (event instanceof KeyboardEvent ? event.key === 'Escape' : !root.current?.contains(event.target as Node)) close();
    };
    window.addEventListener('mousedown', done);
    window.addEventListener('keydown', done);
    return () => { window.removeEventListener('mousedown', done); window.removeEventListener('keydown', done); };
  }, [close]);
  const { buttons, reason } = testAvailability(node.data, project.ieds, running);
  const note = reason === 'real' ? TEST_COPY.real : reason === 'not_running' ? TEST_COPY.notRunning : '';
  return <div ref={root} className="popover test-menu" role="menu" aria-label={`Test ${node.data.label}`}
    style={{ position: 'fixed', left: Math.min(at.x, innerWidth - 240), top: Math.min(at.y, innerHeight - 300), minWidth: 200 }}>
    <p className="small muted test-menu-title">{TEST_COPY.title} · {node.data.label}</p>
    {buttons.map(button => <button key={`${button.row || ''}:${button.action}`} role="menuitem" className="menu-item"
      onClick={() => { close(); if (button.action === 'set') openSheet(); else test(node.id, button.action, button.row ? { row: button.row } : undefined); }}>
      {button.action === 'set' ? `${button.label}…` : button.label}</button>)}
    {note && <p className="small muted test-menu-note">{note}</p>}
    {!buttons.length && !note && <p className="small muted test-menu-note">No test buttons: connect it to an IED signal first.</p>}
    <button role="menuitem" className="menu-item" onClick={() => { close(); openSheet(); }}>Open properties</button>
  </div>;
}
