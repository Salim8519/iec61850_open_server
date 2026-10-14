import type { Diagram, EquipmentKind, EquipmentNode } from '../../api/types';
import { columnOrder, components, wireGraph, type WireGraph } from './graph.ts';
import { arrangeColumns, GAP_X, GRID, rowCentres, type Size } from './position.ts';
import { layerRows } from './rows.ts';
import { sideWires, turnToWires } from './wires.ts';

type Point = { x: number; y: number };
export type TidyResult = { diagram: Diagram; moved: number; groups: number; parked: number };

/** Space between separate drawings, and the shortest column of parked items. */
const GROUP_GAP = 120;
const PARK_GAP_Y = 40;
const MIN_PARK_HEIGHT = 480;
/** Unwired items are parked beside the drawing in this order: panels first. */
const PARK_ORDER: EquipmentKind[] = ['RELAY', 'TRIP_RELAY', 'PUSH_BUTTON', 'SOURCE', 'CT', 'VT', 'CB', 'DS', 'ES', 'JUNCTION'];

/** Size used when the canvas has not measured an item yet (matches the stylesheet). */
export function defaultSize(node: EquipmentNode): Size {
  if (node.data.kind === 'JUNCTION') return { width: 40, height: 40 };
  if (node.data.kind === 'RELAY') return { width: 216, height: 60 + 34 * (node.data.relay_statuses?.length ?? 0) };
  return { width: 108, height: 120 };
}

const snap = (value: number) => Math.round(value / GRID) * GRID;
const snapUp = (value: number) => Math.ceil(value / GRID) * GRID;

/** Lay out each wired group like a single-line diagram; returns item centres and the drawing's bottom. */
function drawGroups(graph: WireGraph, groups: string[][], size: (id: string) => Size) {
  const centre = new Map<string, Point>();
  let left = 0;
  let bottom = 0;
  for (const group of groups) {
    const rows = layerRows(graph, group);
    const xs = arrangeColumns(graph, rows, size);
    const ys = rowCentres(rows, size);
    const edge = Math.min(...group.map(id => xs.get(id)! - size(id).width / 2));
    const shift = snapUp(left - edge);
    let right = left;
    rows.forEach((row, index) => row.blocks.flat().forEach(id => {
      const point = { x: xs.get(id)! + shift, y: ys[index] };
      centre.set(id, point);
      right = Math.max(right, point.x + size(id).width / 2);
      bottom = Math.max(bottom, point.y + size(id).height / 2);
    }));
    left = right + GROUP_GAP;
  }
  return { centre, left: groups.length ? left : 0, bottom };
}

/** Stack unwired items in columns to the right of the drawing, panels first. */
function park(graph: WireGraph, ids: string[], left: number, bottom: number, size: (id: string) => Size,
  centre: Map<string, Point>) {
  const rank = (id: string) => PARK_ORDER.indexOf(graph.node.get(id)!.data.kind);
  // Column by column, matching how they are stacked, so parking again keeps the order.
  const sorted = [...ids].sort((a, b) => rank(a) - rank(b) || columnOrder(graph.node.get(a)!, graph.node.get(b)!));
  const limit = Math.max(bottom, MIN_PARK_HEIGHT);
  let x = snapUp(left);
  let y = 0;
  let width = 0;
  for (const id of sorted) {
    const { width: w, height: h } = size(id);
    if (y > 0 && y + h > limit) { x = snapUp(x + width + GAP_X); y = 0; width = 0; }
    centre.set(id, { x: x + w / 2, y: y + h / 2 });
    y = snapUp(y + h + PARK_GAP_Y);
    width = Math.max(width, w);
  }
}

type Options = { scope?: ReadonlySet<string>; sizes?: ReadonlyMap<string, Size> };

/** One layout pass; returns the input object when nothing would change. */
function arrangeOnce(diagram: Diagram, options: Options) {
  const scope = options.scope && options.scope.size > 1 ? options.scope : new Set(diagram.nodes.map(node => node.id));
  const graph = wireGraph(diagram, scope);
  const size = (id: string) => options.sizes?.get(id) ?? defaultSize(graph.node.get(id)!);
  const groups = components(graph);
  const { centre, left, bottom } = drawGroups(graph, groups, size);
  const loose = graph.ids.filter(id => !graph.adjacent.get(id)!.length);
  park(graph, loose, left, bottom, size, centre);
  const summary = { groups: groups.length, parked: loose.length };
  if (!centre.size) return { diagram, ...summary };
  const corner = (id: string, point: Point) => ({ x: point.x - size(id).width / 2, y: point.y - size(id).height / 2 });
  const before = [...centre.keys()].map(id => graph.node.get(id)!.position);
  const after = [...centre].map(([id, point]) => corner(id, point));
  const dx = snap(Math.min(...before.map(p => p.x)) - Math.min(...after.map(p => p.x)));
  const dy = snap(Math.min(...before.map(p => p.y)) - Math.min(...after.map(p => p.y)));
  const placed = new Map([...centre].map(([id, point]) => [id, { x: point.x + dx, y: point.y + dy }]));
  const edges = sideWires(diagram.edges, placed);
  let changed = edges.some((edge, index) => edge !== diagram.edges[index]);
  const nodes = diagram.nodes.map(node => {
    const point = placed.get(node.id);
    if (!point) return node;
    const topLeft = corner(node.id, point);
    const position = { x: Math.round(topLeft.x), y: Math.round(topLeft.y) };
    const rotation = turnToWires(node, edges);
    if (position.x === node.position.x && position.y === node.position.y && rotation === (node.data.rotation || 0)) return node;
    changed = true;
    return { ...node, position, data: rotation === node.data.rotation ? node.data : { ...node.data, rotation } };
  });
  return { diagram: changed ? { ...diagram, nodes, edges } : diagram, ...summary };
}

/** Passes until the layout is settled, so pressing Tidy up again changes nothing. */
const MAX_PASSES = 4;

/**
 * Arrange a diagram like a real single-line diagram: busbars across the top, bays
 * hanging straight down, wires straight, and unwired items (relay panels, buttons)
 * parked to the right. Only positions, wire sides and symbol turns change. With a
 * scope of two or more items, only those items move, anchored where they were.
 */
export function tidyDiagram(diagram: Diagram, options: Options = {}): TidyResult {
  let current = diagram;
  let pass = arrangeOnce(current, options);
  for (let count = 1; count < MAX_PASSES && pass.diagram !== current; count += 1) {
    current = pass.diagram;
    pass = arrangeOnce(current, options);
  }
  const original = new Map(diagram.nodes.map(node => [node.id, node]));
  const moved = pass.diagram.nodes.filter(node => {
    const before = original.get(node.id)!;
    return before.position.x !== node.position.x || before.position.y !== node.position.y ||
      (before.data.rotation || 0) !== (node.data.rotation || 0);
  }).length;
  const rewired = pass.diagram.edges.some((edge, index) => {
    const before = diagram.edges[index];
    return edge.source !== before.source || edge.target !== before.target ||
      edge.sourceHandle !== before.sourceHandle || edge.targetHandle !== before.targetHandle;
  });
  return { diagram: moved || rewired ? pass.diagram : diagram, moved, groups: pass.groups, parked: pass.parked };
}
