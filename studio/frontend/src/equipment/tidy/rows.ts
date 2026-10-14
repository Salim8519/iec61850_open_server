import { barNeighbours, busbars, distances, SHARED_BAY_WIRES, topEnd, type WireGraph } from './graph.ts';

/**
 * One horizontal line of the tidied drawing, as blocks: a busbar is one block that
 * stays together in bar order; any other item is a block of its own. `bars` marks a
 * row of busbars only, which stacks tighter under another such row.
 */
export type Row = { blocks: string[][]; bars: boolean };

/**
 * The main busbar is on top. Another bar is stacked under it when two or more of its
 * bays also reach a top bar through a neighbouring bay: a double busbar or bus coupler.
 */
function topBars(graph: WireGraph, bars: string[][]): number[] {
  if (!bars.length) return [];
  const top = [0];
  for (let index = 1; index < bars.length; index += 1) {
    const reach = distances(graph, top.flatMap(bar => bars[bar]), new Set(bars[index]));
    const [near, far] = SHARED_BAY_WIRES;
    const shared = barNeighbours(graph, bars[index]).filter(id => {
      const wires = reach.get(id) ?? Infinity;
      return wires >= near && wires <= far;
    });
    if (shared.length >= 2) top.push(index);
  }
  return top;
}

/**
 * Rows of one wired group, top to bottom: busbars first, then every item at its wire
 * distance from them, like bays hanging from a bar. A whole busbar always shares one row.
 */
export function layerRows(graph: WireGraph, group: string[]): Row[] {
  const bars = busbars(graph, group);
  const barOf = new Map<string, number>();
  bars.forEach((bar, index) => bar.forEach(id => barOf.set(id, index)));
  const rank = new Map<string, number>();
  const queue: string[] = [];
  const seed = (ids: string[], value: number) => {
    for (const id of ids) if (!rank.has(id)) { rank.set(id, value); queue.push(id); }
  };
  const top = topBars(graph, bars);
  if (top.length) top.forEach(bar => seed(bars[bar], 0));
  else seed([topEnd(graph, group)], 0);
  for (let index = 0; index < queue.length; index += 1) {
    const current = queue[index];
    for (const next of graph.adjacent.get(current)!) {
      if (rank.has(next)) continue;
      const value = rank.get(current)! + 1;
      seed(barOf.has(next) ? bars[barOf.get(next)!] : [next], value);
    }
  }
  const deepest = Math.max(...rank.values());
  const rows: Row[] = top.map(bar => ({ blocks: [[...bars[bar]]], bars: true }));
  const placed = new Set(top.flatMap(bar => bars[bar]));
  for (let level = 0; level <= deepest; level += 1) {
    // Blocks start in the order the walk from the busbar met them, so bays stay together.
    const blocks: string[][] = [];
    for (const id of queue) {
      if (rank.get(id) !== level || placed.has(id)) continue;
      const block = barOf.has(id) ? [...bars[barOf.get(id)!]] : [id];
      block.forEach(member => placed.add(member));
      blocks.push(block);
    }
    if (blocks.length) rows.push({ blocks, bars: blocks.every(block => block.length > 1) });
  }
  return rows;
}
