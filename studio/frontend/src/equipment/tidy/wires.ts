import type { Diagram, EquipmentKind, EquipmentNode } from '../../api/types';
import { connectedSides } from '../junctionSides.ts';

type Point = { x: number; y: number };
type Edge = Diagram['edges'][number];

/** Symbols drawn along their wire; turning them follows the wire direction. */
const TURNABLE: ReadonlySet<EquipmentKind> = new Set(['CB', 'DS', 'ES', 'CT', 'VT', 'SOURCE', 'TRIP_RELAY']);

/**
 * Re-side wires between tidied items so smooth-step lines run straight: items in one
 * row join right side to left side, items in different rows join bottom to top.
 * Wire ids and the two items a wire joins never change.
 */
export function sideWires(edges: Edge[], centre: ReadonlyMap<string, Point>): Edge[] {
  return edges.map(edge => {
    const a = centre.get(edge.source);
    const b = centre.get(edge.target);
    if (!a || !b || edge.source === edge.target) return edge;
    let ends: [string, string, string, string];
    if (Math.abs(a.y - b.y) < 1) {
      ends = a.x <= b.x ? [edge.source, 'east', edge.target, 'west'] : [edge.target, 'east', edge.source, 'west'];
    } else {
      ends = a.y < b.y ? [edge.source, 'south', edge.target, 'north'] : [edge.target, 'south', edge.source, 'north'];
    }
    const [source, sourceHandle, target, targetHandle] = ends;
    if (edge.source === source && edge.target === target && edge.sourceHandle === sourceHandle &&
        edge.targetHandle === targetHandle) return edge;
    return { ...edge, source, sourceHandle, target, targetHandle };
  });
}

/** Upright symbols on vertical wires, turned a quarter on horizontal wires; mixed wiring keeps its turn. */
export function turnToWires(node: EquipmentNode, edges: Edge[]): number {
  const rotation = node.data.rotation || 0;
  if (!TURNABLE.has(node.data.kind)) return rotation;
  const sides = connectedSides(node.id, edges);
  if (!sides.size) return rotation;
  const vertical = [...sides].every(side => side === 'north' || side === 'south');
  const horizontal = [...sides].every(side => side === 'east' || side === 'west');
  if (vertical && rotation % 180 !== 0) return 0;
  if (horizontal && rotation % 180 === 0) return 90;
  return rotation;
}
