import { columnOrder, type WireGraph } from './graph.ts';
import type { Row } from './rows.ts';

export type Size = { width: number; height: number };

export const GRID = 20;
/** Clear space between neighbours in a row, and between rows. */
export const GAP_X = 44;
const GAP_Y = 60;
const GAP_Y_BARS = 40;
const SWEEPS = 10;
const ORDERING_SWEEPS = SWEEPS - 2;
/**
 * How strongly an item holds its column: a run of equipment (wired on both sides) beats
 * an in-line dead end (feeder end), which beats a side branch (earth switch, VT).
 */
const WEIGHT = { run: 1000, end: 30, branch: 1 };
const BRANCH_KINDS = new Set(['ES', 'VT']);

/**
 * Positions nearest to `wanted` that keep the given order and minimum distances:
 * weighted isotonic regression (pool adjacent violators) on gap-shifted targets.
 */
export function fitInOrder(wanted: number[], gaps: number[], weights: number[] = wanted.map(() => 1)): number[] {
  const offset = [0];
  gaps.forEach((gap, index) => offset.push(offset[index] + gap));
  const blocks: { total: number; weight: number; count: number }[] = [];
  wanted.forEach((value, index) => {
    blocks.push({ total: (value - offset[index]) * weights[index], weight: weights[index], count: 1 });
    while (blocks.length > 1) {
      const last = blocks[blocks.length - 1];
      const before = blocks[blocks.length - 2];
      if (before.total / before.weight <= last.total / last.weight) break;
      blocks.pop();
      before.total += last.total; before.weight += last.weight; before.count += last.count;
    }
  });
  const result: number[] = [];
  for (const block of blocks) {
    for (let step = 0; step < block.count; step += 1) result.push(block.total / block.weight + offset[result.length]);
  }
  return result;
}

function median(values: number[]): number {
  const sorted = [...values].sort((a, b) => a - b);
  const middle = Math.floor(sorted.length / 2);
  return sorted.length % 2 ? sorted[middle] : (sorted[middle - 1] + sorted[middle]) / 2;
}

const snap = (value: number) => Math.round(value / GRID) * GRID;
const snapUp = (value: number) => Math.ceil(value / GRID) * GRID;

/**
 * Centre x of every item. Sweeps alternate down and up: each item is pulled under
 * (or over) the items it is wired to, rows are re-ordered to remove crossings, and the
 * last two sweeps only straighten. Dead-end branches never bend a run of equipment.
 */
export function arrangeColumns(graph: WireGraph, sourceRows: Row[], size: (id: string) => Size): Map<string, number> {
  const rows = sourceRows.map(row => row.blocks.map(block => [...block]));
  const rowOf = new Map<string, number>();
  rows.forEach((blocks, index) => blocks.flat().forEach(id => rowOf.set(id, index)));
  const x = new Map<string, number>();
  const role = (id: string): keyof typeof WEIGHT => graph.adjacent.get(id)!.length > 1 ? 'run' :
    BRANCH_KINDS.has(graph.node.get(id)!.data.kind) ? 'branch' : 'end';
  const gaps = (ids: string[]) => ids.slice(1).map((id, index) => (size(ids[index]).width + size(id).width) / 2 + GAP_X);
  const place = (ids: string[], wanted: number[], snapped: boolean) => {
    const xs = fitInOrder(wanted, gaps(ids), ids.map(id => WEIGHT[role(id)]));
    ids.forEach((id, index) => x.set(id, snapped ? snap(xs[index]) : xs[index]));
  };
  rows.forEach(blocks => place(blocks.flat(), blocks.flat().map(() => 0), false));
  const pulls = (id: string, row: number, down: boolean) => {
    const wired = graph.adjacent.get(id)!.filter(other => down ? rowOf.get(other)! < row : rowOf.get(other)! > row);
    const strongest = Math.max(...wired.map(other => WEIGHT[role(other)]));
    return wired.filter(other => WEIGHT[role(other)] === strongest).map(other => x.get(other)!);
  };
  const misfit = (ids: string[], want: number[]) =>
    fitInOrder(want, gaps(ids)).reduce((sum, value, index) => sum + Math.abs(value - want[index]), 0);
  for (let sweep = 0; sweep < SWEEPS; sweep += 1) {
    const down = sweep % 2 === 0;
    for (let step = 0; step < rows.length; step += 1) {
      const index = down ? step : rows.length - 1 - step;
      const target = (id: string) => { const from = pulls(id, index, down); return from.length ? median(from) : x.get(id)!; };
      if (sweep < ORDERING_SWEEPS) {
        // A bar faces the way its bays hang; blocks then sort by where they are pulled.
        rows[index] = rows[index].map(block => {
          if (block.length < 2) return block;
          const flipped = [...block].reverse();
          return misfit(flipped, flipped.map(target)) < misfit(block, block.map(target)) ? flipped : block;
        });
        // Equal pulls keep the user's left-to-right drawing order, so tidying twice changes nothing.
        const key = new Map(rows[index].map(block => [block, block.reduce((sum, id) => sum + target(id), 0) / block.length]));
        rows[index].sort((a, b) => Math.abs(key.get(a)! - key.get(b)!) > 0.5 ? key.get(a)! - key.get(b)! :
          columnOrder(graph.node.get(a[0])!, graph.node.get(b[0])!));
      }
      const ids = rows[index].flat();
      place(ids, ids.map(target), sweep === SWEEPS - 1);
    }
  }
  // Junction rows sit exactly under what feeds them; spacing may have nudged the row above.
  sourceRows.forEach((row, index) => {
    if (!row.bars || !index) return;
    const ids = rows[index].flat();
    place(ids, ids.map(id => { const from = pulls(id, index, true); return from.length ? median(from) : x.get(id)!; }), true);
  });
  return x;
}

/** Centre y of every row: equal clear space between rows, tighter between stacked bars. */
export function rowCentres(rows: Row[], size: (id: string) => Size): number[] {
  const heights = rows.map(row => Math.max(...row.blocks.flat().map(id => size(id).height)));
  const centres: number[] = [];
  heights.forEach((height, index) => {
    if (!index) { centres.push(snapUp(height / 2)); return; }
    const gap = rows[index].bars && rows[index - 1].bars ? GAP_Y_BARS : GAP_Y;
    centres.push(snapUp(centres[index - 1] + heights[index - 1] / 2 + gap + height / 2));
  });
  return centres;
}
