import type { Diagram, EquipmentNode } from '../../api/types';

/** Undirected wiring between the items being tidied, with neighbours in a fixed order. */
export type WireGraph = {
  ids: string[];
  node: Map<string, EquipmentNode>;
  adjacent: Map<string, string[]>;
};

/** A junction with this many wires is drawn as a busbar point rather than a tee. */
export const BUS_POINT_WIRES = 4;
/**
 * A bar whose bays reach a top bar through another bay (2-3 wires: disconnector, tee,
 * disconnector) is a second busbar. A bar right under a top bar (1 wire) is a bay tee.
 */
export const SHARED_BAY_WIRES = [2, 3];

/** Reading order of the user's drawing: top to bottom, then left to right, then id. */
export function readingOrder(a: EquipmentNode, b: EquipmentNode): number {
  return a.position.y - b.position.y || a.position.x - b.position.x || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0);
}

/** Left to right, then top to bottom: the order bays were drawn in. */
export function columnOrder(a: EquipmentNode, b: EquipmentNode): number {
  return a.position.x - b.position.x || a.position.y - b.position.y || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0);
}

export function wireGraph(diagram: Diagram, scope: ReadonlySet<string>): WireGraph {
  const node = new Map(diagram.nodes.filter(item => scope.has(item.id)).map(item => [item.id, item]));
  const sets = new Map([...node.keys()].map(id => [id, new Set<string>()]));
  for (const edge of diagram.edges) {
    if (edge.source === edge.target || !node.has(edge.source) || !node.has(edge.target)) continue;
    sets.get(edge.source)!.add(edge.target);
    sets.get(edge.target)!.add(edge.source);
  }
  const byColumn = (a: string, b: string) => columnOrder(node.get(a)!, node.get(b)!);
  const adjacent = new Map([...sets].map(([id, set]) => [id, [...set].sort(byColumn)]));
  const ids = [...node.keys()].sort((a, b) => readingOrder(node.get(a)!, node.get(b)!));
  return { ids, node, adjacent };
}

/** Wired groups (two or more items), ordered left to right as drawn. */
export function components(graph: WireGraph): string[][] {
  const seen = new Set<string>();
  const result: string[][] = [];
  for (const start of graph.ids) {
    if (seen.has(start) || !graph.adjacent.get(start)!.length) continue;
    const group = [start];
    seen.add(start);
    for (let index = 0; index < group.length; index += 1) {
      for (const next of graph.adjacent.get(group[index])!) if (!seen.has(next)) { seen.add(next); group.push(next); }
    }
    result.push(group);
  }
  const left = (group: string[]) => Math.min(...group.map(id => graph.node.get(id)!.position.x));
  return result.sort((a, b) => left(a) - left(b) || readingOrder(graph.node.get(a[0])!, graph.node.get(b[0])!));
}

const isJunction = (graph: WireGraph, id: string) => graph.node.get(id)!.data.kind === 'JUNCTION';

/** Walk a junction cluster end to end when it is a simple chain; otherwise order it as drawn. */
function chainOrder(graph: WireGraph, cluster: string[]): string[] {
  const inside = new Set(cluster);
  const links = (id: string) => graph.adjacent.get(id)!.filter(other => inside.has(other));
  const ends = cluster.filter(id => links(id).length === 1);
  const isChain = ends.length === 2 && cluster.every(id => links(id).length <= 2);
  if (!isChain) return [...cluster].sort((a, b) => columnOrder(graph.node.get(a)!, graph.node.get(b)!));
  const [first, last] = ends.sort((a, b) => columnOrder(graph.node.get(a)!, graph.node.get(b)!));
  const chain = [first];
  while (chain[chain.length - 1] !== last) {
    chain.push(links(chain[chain.length - 1]).find(id => !chain.includes(id))!);
  }
  return chain;
}

/** Items wired to a bar from outside it. */
export function barNeighbours(graph: WireGraph, bar: string[]): string[] {
  const own = new Set(bar);
  return [...new Set(bar.flatMap(id => graph.adjacent.get(id)!.filter(other => !own.has(other))))];
}

/**
 * Busbars: junctions wired to each other form one bar; a lone junction with many
 * wires is a bar point. The bar with the most connections comes first (the main
 * busbar), ties as drawn top to bottom; each bar is in bar order.
 */
export function busbars(graph: WireGraph, group: string[]): string[][] {
  const seen = new Set<string>();
  const bars: string[][] = [];
  for (const start of group) {
    if (seen.has(start) || !isJunction(graph, start)) continue;
    const cluster = [start];
    seen.add(start);
    for (let index = 0; index < cluster.length; index += 1) {
      for (const next of graph.adjacent.get(cluster[index])!) {
        if (!seen.has(next) && isJunction(graph, next)) { seen.add(next); cluster.push(next); }
      }
    }
    if (cluster.length > 1 || graph.adjacent.get(start)!.length >= BUS_POINT_WIRES) bars.push(chainOrder(graph, cluster));
  }
  const top = (bar: string[]) => Math.min(...bar.map(id => graph.node.get(id)!.position.y));
  const reach = new Map(bars.map(bar => [bar, barNeighbours(graph, bar).length]));
  return bars.sort((a, b) => reach.get(b)! - reach.get(a)! || b.length - a.length || top(a) - top(b) ||
    readingOrder(graph.node.get(a[0])!, graph.node.get(b[0])!));
}

/**
 * Wire-count distance from a set of items to every item of the group, never passing
 * `blocked`. Wires between two junctions are free: a chain of junctions is one point.
 */
export function distances(graph: WireGraph, from: string[], blocked: ReadonlySet<string> = new Set()): Map<string, number> {
  const result = new Map(from.map(id => [id, 0]));
  const queue = [...from];
  for (let index = 0; index < queue.length; index += 1) {
    const current = queue[index];
    for (const next of graph.adjacent.get(current)!) {
      if (blocked.has(next)) continue;
      const cost = result.get(current)! + (isJunction(graph, current) && isJunction(graph, next) ? 0 : 1);
      if (cost < (result.get(next) ?? Infinity)) { result.set(next, cost); queue.push(next); }
    }
  }
  return result;
}

/** Where a group without busbars starts: the end of its longest run that was drawn highest. */
export function topEnd(graph: WireGraph, group: string[]): string {
  const farthest = (from: string) => {
    const reach = distances(graph, [from]);
    return group.reduce((best, id) => reach.get(id)! > reach.get(best)! ? id : best, group[0]);
  };
  const one = farthest(group[0]);
  const two = farthest(one);
  return readingOrder(graph.node.get(one)!, graph.node.get(two)!) <= 0 ? one : two;
}
