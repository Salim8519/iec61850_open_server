import type { GooseProposal } from '../../../api/gooseMessages';
import type { Diagram, EquipmentNode } from '../../../api/types';

/** IED components placed where Go proposed them (ADR-0112). Pure: the browser only turns Go's
 * placements into diagram nodes; one edit adds them all, so one Undo removes them all. */
export type Placement = GooseProposal['nodes'][number];

export function iedComponent(place: Placement, id: string): EquipmentNode {
  return { id, type: 'equipment', position: { x: place.position.x, y: place.position.y },
    data: { kind: 'IED', label: place.name, rotation: 0, ied_id: place.ied_id } };
}

/** The drawing with one IED component per placement; IEDs already drawn are skipped. */
export function withIedComponents(diagram: Diagram, places: Placement[], makeId: () => string) {
  const drawn = new Set(diagram.nodes.filter(node => node.data.kind === 'IED').map(node => node.data.ied_id));
  const nodes = places.filter(place => !drawn.has(place.ied_id)).map(place => iedComponent(place, makeId()));
  return { diagram: nodes.length ? { ...diagram, nodes: [...diagram.nodes, ...nodes] } : diagram, ids: nodes.map(node => node.id) };
}

/** One IED from Go's list, put where the next component goes (the first proposed slot), so adding
 * them one by one continues the same row. */
export function nextSlot(places: Placement[], iedId: string): Placement[] {
  const place = places.find(item => item.ied_id === iedId);
  return place && places.length ? [{ ...place, position: places[0].position }] : [];
}
