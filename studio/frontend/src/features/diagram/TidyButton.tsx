import { useEffect, useState } from 'react';
import { useReactFlow } from '@xyflow/react';
import { WandSparkles } from 'lucide-react';
import type { Diagram, EquipmentNode } from '../../api/types';
import { tidyDiagram } from '../../equipment/tidy/tidy';
import { tidyMessage, tidyTitle } from './tidyCopy';

const NOTE_MS = 6000;

/**
 * "Tidy up": arrange the drawing like a standard single-line diagram in one undo step.
 * Rendered inside the canvas so it can use the measured size of every symbol.
 */
export function TidyButton({ diagram, selection, locked, apply }: {
  diagram: Diagram; selection: string[]; locked: boolean; apply: (diagram: Diagram) => void;
}) {
  const flow = useReactFlow<EquipmentNode>();
  const [note, setNote] = useState('');
  useEffect(() => {
    if (!note) return;
    const timer = window.setTimeout(() => setNote(''), NOTE_MS);
    return () => window.clearTimeout(timer);
  }, [note]);
  const scoped = selection.length > 1;
  const run = () => {
    const sizes = new Map(flow.getNodes().flatMap(node => node.measured?.width && node.measured.height ?
      [[node.id, { width: node.measured.width, height: node.measured.height }] as const] : []));
    const result = tidyDiagram(diagram, { scope: scoped ? new Set(selection) : undefined, sizes });
    const changed = result.diagram !== diagram;
    setNote(tidyMessage(changed, result.moved, scoped));
    if (!changed) return;
    apply(result.diagram);
    const still = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    window.setTimeout(() => flow.fitView({ maxZoom: 1, padding: 0.2, duration: still ? 0 : 300,
      nodes: scoped ? selection.map(id => ({ id })) : undefined }), 60);
  };
  return <>
    <button className="ghost" disabled={locked || diagram.nodes.length < 2} onClick={run}
      title={tidyTitle(locked, scoped)}><WandSparkles size={17} />{scoped ? 'Tidy selection' : 'Tidy up'}</button>
    {note && <span className="tidy-note" role="status">{note}</span>}
  </>;
}
