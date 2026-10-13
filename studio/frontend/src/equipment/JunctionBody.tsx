import { useNodeConnections } from '@xyflow/react';
import { connectedSides, type Side } from './junctionSides';

const STUB: Record<Side, [number, number]> = { north: [18, 0], east: [36, 18], south: [18, 36], west: [0, 18] };

/** Junction dot with stubs to wired terminals; the rest of the box is the drag grip. */
export function JunctionBody({ nodeId }: { nodeId: string }) {
  const sides = connectedSides(nodeId, useNodeConnections());
  return <svg width={36} height={36} viewBox="0 0 36 36" aria-hidden="true">
    {[...sides].map(side => <line key={side} x1={18} y1={18} x2={STUB[side][0]} y2={STUB[side][1]}
      stroke="currentColor" strokeWidth={2} />)}
    <circle cx={18} cy={18} r={5} fill="currentColor" />
  </svg>;
}
