import { useMemo } from 'react';
import { useReactFlow } from '@xyflow/react';
import type { Diagram } from '../../api/types';
import { besideDrawing, strayIds } from '../../equipment/strayNodes';

/** Names symbols far away from the drawing (change 0180) and moves them next to it in one Undo step. */
export function StrayNotice({ diagram, locked, change }: { diagram: Diagram; locked: boolean;
  change: (diagram: Diagram) => void }) {
  const flow = useReactFlow();
  const strays = useMemo(() => strayIds(diagram.nodes), [diagram.nodes]);
  if (!strays.length) return null;
  const names = diagram.nodes.filter(node => strays.includes(node.id)).map(node => node.data.label || node.data.kind);
  const bring = () => {
    const at = besideDrawing(diagram.nodes, strays);
    let column = 0;
    change({ ...diagram, nodes: diagram.nodes.map(node => strays.includes(node.id)
      ? { ...node, position: { x: at.x + 160 * column++, y: at.y } } : node) });
    window.setTimeout(() => flow.fitView({ maxZoom: 1, padding: 0.3, duration: 300 }), 60);
  };
  return <div className="stray-notice" role="status">
    <span>{names.length === 1 ? `${names[0]} is` : `${names.length} symbols (${names.join(', ')}) are`} far away from the
      drawing, so the view leaves {names.length === 1 ? 'it' : 'them'} out.</span>
    {!locked && <button type="button" className="secondary" onClick={bring}>Bring {names.length === 1 ? 'it' : 'them'} here</button>}
  </div>;
}
