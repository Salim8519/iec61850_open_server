import type { Diagram, EquipmentNode } from '../api/types';

/** Copied canvas content: nodes plus only the wires whose both ends were copied. */
export type Clip = { nodes: EquipmentNode[]; edges: Diagram['edges'] };
export const PASTE_OFFSET = 40;

export function copySelection(diagram: Diagram, ids: ReadonlySet<string>): Clip | null {
  const nodes = diagram.nodes.filter(node => ids.has(node.id));
  if (!nodes.length) return null;
  const edges = diagram.edges.filter(edge => ids.has(edge.source) && ids.has(edge.target));
  return structuredClone({
    nodes: nodes.map(({ id, type, position, data }) => ({ id, type, position, data })),
    edges: edges.map(({ id, source, target, sourceHandle, targetHandle, type }) =>
      ({ id, source, target, sourceHandle, targetHandle, type })),
  });
}

/** Next free "<prefix>-<n>" label, or "<label> copy[ n]" when the label has no number. */
export function uniqueLabel(label: string, taken: Set<string>): string {
  const numbered = /^(.*?)(\d+)$/.exec(label);
  if (numbered) {
    for (let n = Number(numbered[2]) + 1; ; n += 1) {
      const candidate = `${numbered[1]}${n}`;
      if (!taken.has(candidate)) return candidate;
    }
  }
  if (!taken.has(`${label} copy`)) return `${label} copy`;
  for (let n = 2; ; n += 1) if (!taken.has(`${label} copy ${n}`)) return `${label} copy ${n}`;
}

/** Insert a clip with fresh identifiers, shifted by `offset`; returns the new node IDs. */
export function pasteClip(diagram: Diagram, clip: Clip, offset: number,
  newId: () => string = () => crypto.randomUUID()): { diagram: Diagram; ids: string[] } {
  const taken = new Set(diagram.nodes.map(node => node.data.label));
  const idMap = new Map<string, string>();
  const nodes = clip.nodes.map(node => {
    const id = newId();
    idMap.set(node.id, id);
    const label = uniqueLabel(node.data.label, taken);
    taken.add(label);
    // A copied IED component starts without its IED: one component per IED (ADR-0112).
    const data = node.data.kind === 'IED' ? { ...structuredClone(node.data), label, ied_id: null } : { ...structuredClone(node.data), label };
    return { ...structuredClone(node), id, position: { x: node.position.x + offset, y: node.position.y + offset }, data };
  });
  const edges = clip.edges.map(edge => ({ ...edge, id: `e-${newId()}`,
    source: idMap.get(edge.source)!, target: idMap.get(edge.target)! }));
  return { diagram: { nodes: [...diagram.nodes, ...nodes], edges: [...diagram.edges, ...edges] },
    ids: nodes.map(node => node.id) };
}

/** Remove nodes and every wire attached to them. */
export function removeNodes(diagram: Diagram, ids: ReadonlySet<string>): Diagram {
  return { nodes: diagram.nodes.filter(node => !ids.has(node.id)),
    edges: diagram.edges.filter(edge => !ids.has(edge.source) && !ids.has(edge.target)) };
}

export function nudgeNodes(diagram: Diagram, ids: ReadonlySet<string>, dx: number, dy: number): Diagram {
  return { ...diagram, nodes: diagram.nodes.map(node => ids.has(node.id) ?
    { ...node, position: { x: node.position.x + dx, y: node.position.y + dy } } : node) };
}
