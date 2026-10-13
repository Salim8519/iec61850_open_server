import type { Diagram } from '../api/types';

export type DiagramHistory = { past: Diagram[]; present: Diagram; future: Diagram[]; group?: string };
const LIMIT = 50;

function engineering(diagram: Diagram): Diagram {
  return { nodes: diagram.nodes.map(({ id, type, position, data }) => ({ id, type, position, data })),
    edges: diagram.edges.map(({ id, source, target, sourceHandle, targetHandle }) =>
      ({ id, source, target, sourceHandle, targetHandle })) };
}

/** True when two diagrams persist identically (view-only fields such as measured sizes ignored). */
export function sameEngineering(a: Diagram, b: Diagram): boolean {
  return JSON.stringify(engineering(a)) === JSON.stringify(engineering(b));
}

export function createHistory(diagram: Diagram): DiagramHistory {
  return { past: [], present: structuredClone(engineering(diagram)), future: [] };
}

export function recordDiagram(history: DiagramHistory, diagram: Diagram, group?: string): DiagramHistory {
  const next = engineering(diagram);
  if (JSON.stringify(next) === JSON.stringify(history.present)) return history;
  return { past: group && history.group === group ? history.past :
    [...history.past, history.present].slice(-LIMIT), present: structuredClone(next), future: [], group };
}

export function undoDiagram(history: DiagramHistory): DiagramHistory {
  if (!history.past.length) return history;
  return { past: history.past.slice(0, -1), present: history.past[history.past.length - 1],
    future: [history.present, ...history.future] };
}

export function redoDiagram(history: DiagramHistory): DiagramHistory {
  if (!history.future.length) return history;
  return { past: [...history.past, history.present], present: history.future[0], future: history.future.slice(1) };
}
