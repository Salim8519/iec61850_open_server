import type { Diagram, EquipmentData, EquipmentKind, EquipmentNode } from '../api/types';
import type { Clip } from './clipboard.ts';

/** Ready-made bays (spec bay-templates). Drawing only: nothing is linked to an IED yet. */
export type BayKind = 'line_feeder' | 'transformer_bay' | 'bus_coupler';
type Part = { id: string; kind: EquipmentKind; code: string; col: number; row: number; extra?: Partial<EquipmentData> };
type Bay = { name: string; prefix: string; hint: string; parts: Part[]; wires: [string, string, 'down' | 'across'][];
  anchors: string[] };

/** Same spacing as Tidy up: 160 px columns, 180 px rows for symbols. */
const COLUMN = 160;
const ROW = 180;
const CARD = { width: 108, height: 120 };
const DOT = 40;

export const BAYS: Record<BayKind, Bay> = {
  line_feeder: {
    name: 'Line feeder', prefix: 'F', hint: 'Bus disconnector, breaker, CT, line disconnector, earth switch and VT',
    parts: [
      { id: 'q1', kind: 'DS', code: 'Q1', col: 0, row: 0 }, { id: 'q0', kind: 'CB', code: 'Q0', col: 0, row: 1 },
      { id: 't1', kind: 'CT', code: 'T1', col: 0, row: 2 }, { id: 'q9', kind: 'DS', code: 'Q9', col: 0, row: 3 },
      { id: 'j', kind: 'JUNCTION', code: 'J', col: 0, row: 3.67 },
      { id: 'q8', kind: 'ES', code: 'Q8', col: 1, row: 4.45 }, { id: 't5', kind: 'VT', code: 'T5', col: -1, row: 4.45 },
      { id: 'x', kind: 'INFEED', code: 'Line', col: 0, row: 4.45, extra: { infeed_type: 'line', infeed_state: 'dead' } },
    ],
    wires: [['q1', 'q0', 'down'], ['q0', 't1', 'down'], ['t1', 'q9', 'down'], ['q9', 'j', 'down'],
      ['j', 'x', 'down'], ['j', 'q8', 'down'], ['j', 't5', 'down']],
    anchors: ['q1'],
  },
  transformer_bay: {
    name: 'Transformer bay', prefix: 'T', hint: 'Bus disconnector, breaker, CT, earth switch and the transformer',
    parts: [
      { id: 'q1', kind: 'DS', code: 'Q1', col: 0, row: 0 }, { id: 'q0', kind: 'CB', code: 'Q0', col: 0, row: 1 },
      { id: 't1', kind: 'CT', code: 'T1', col: 0, row: 2 }, { id: 'j', kind: 'JUNCTION', code: 'J', col: 0, row: 2.67 },
      { id: 'q8', kind: 'ES', code: 'Q8', col: 1, row: 3.45 },
      { id: 'x', kind: 'INFEED', code: 'Transformer', col: 0, row: 3.45, extra: { infeed_type: 'transformer', infeed_state: 'live' } },
    ],
    wires: [['q1', 'q0', 'down'], ['q0', 't1', 'down'], ['t1', 'j', 'down'], ['j', 'x', 'down'], ['j', 'q8', 'down']],
    anchors: ['q1'],
  },
  bus_coupler: {
    name: 'Bus coupler', prefix: 'C', hint: 'Joins two busbars: disconnector, breaker, disconnector',
    parts: [
      { id: 'q1', kind: 'DS', code: 'Q1', col: 0, row: 0 }, { id: 'j1', kind: 'JUNCTION', code: 'JA', col: 0, row: 0.85 },
      { id: 'q0', kind: 'CB', code: 'Q0', col: 1, row: 0.85, extra: { rotation: 90 } },
      { id: 'j2', kind: 'JUNCTION', code: 'JB', col: 2, row: 0.85 }, { id: 'q2', kind: 'DS', code: 'Q2', col: 2, row: 0 },
    ],
    wires: [['q1', 'j1', 'down'], ['j1', 'q0', 'across'], ['q0', 'j2', 'across'], ['q2', 'j2', 'down']],
    anchors: ['q1', 'q2'],
  },
};
export const BAY_ORDER: BayKind[] = ['line_feeder', 'transformer_bay', 'bus_coupler'];

/** Next free bay number for a prefix, so labels read F1-Q0, F2-Q0, ... */
export function nextBayNumber(diagram: Diagram, prefix: string): number {
  const used = new Set(diagram.nodes.map(node => new RegExp(`^${prefix}(\\d+)-`).exec(node.data.label)?.[1]).filter(Boolean).map(Number));
  let number = 1;
  while (used.has(number)) number += 1;
  return number;
}

function clip(kind: BayKind, number: number): Clip {
  const bay = BAYS[kind];
  const nodes: EquipmentNode[] = bay.parts.map(part => {
    const size = part.kind === 'JUNCTION' ? { width: DOT, height: DOT } : CARD;
    const data: EquipmentData = { kind: part.kind, label: `${bay.prefix}${number}-${part.code}`, rotation: 0,
      ...(['CB', 'DS', 'ES'].includes(part.kind) ? { initial_position: 'open' as const } : {}), ...part.extra };
    return { id: part.id, type: 'equipment', data,
      position: { x: Math.round(part.col * COLUMN - size.width / 2), y: Math.round(part.row * ROW + (CARD.height - size.height) / 2) } };
  });
  const edges = bay.wires.map(([source, target, way]) => ({ id: `${source}-${target}`, source, target, type: 'smoothstep',
    sourceHandle: way === 'down' ? 'south' : 'east', targetHandle: way === 'down' ? 'north' : 'west' }));
  return { nodes, edges };
}

/**
 * Add a bay as one edit. With busbar junctions given, the bay's top disconnector(s)
 * are wired to them and the bay hangs under the first one; otherwise it goes at `at`.
 */
export function addBay(diagram: Diagram, kind: BayKind, options: { at?: { x: number; y: number }; attach?: string[] } = {},
  newId: () => string = () => crypto.randomUUID()) {
  const bay = BAYS[kind];
  const content = clip(kind, nextBayNumber(diagram, bay.prefix));
  const bars = (options.attach ?? []).map(id => diagram.nodes.find(node => node.id === id))
    .filter((node): node is EquipmentNode => node?.data.kind === 'JUNCTION').slice(0, bay.anchors.length);
  const first = content.nodes.find(node => node.id === bay.anchors[0])!;
  const target = bars[0] ? { x: bars[0].position.x + DOT / 2 - CARD.width / 2, y: bars[0].position.y + DOT + 60 } :
    options.at ?? { x: 200, y: 100 };
  const dx = target.x - first.position.x, dy = target.y - first.position.y;
  content.nodes.forEach(node => { node.position = { x: node.position.x + dx, y: node.position.y + dy }; });
  // Fresh ids like a paste, but labels stay F<n>-Q0 style (the bay number is already free).
  const local = new Map(content.nodes.map(node => [node.id, newId()]));
  const nodes = content.nodes.map(node => ({ ...node, id: local.get(node.id)! }));
  const wires = content.edges.map(edge => ({ ...edge, id: `e-${newId()}`, source: local.get(edge.source)!, target: local.get(edge.target)! }));
  const links = bars.map((bar, index) => ({ id: `e-${newId()}`, source: bar.id, sourceHandle: 'south', type: 'smoothstep',
    target: local.get(bay.anchors[index])!, targetHandle: 'north' }));
  return { diagram: { nodes: [...diagram.nodes, ...nodes], edges: [...diagram.edges, ...wires, ...links] },
    ids: nodes.map(node => node.id) };
}
