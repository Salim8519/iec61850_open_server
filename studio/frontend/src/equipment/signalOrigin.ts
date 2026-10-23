import type { EquipmentKind, EquipmentNode } from '../api/types';

/** Which SLD component triggered a changed IED signal (ADR-0112, refining ADR-0103): only a
 * component the user acted on, never a guess. `candidates` are the equipment whose own bindings
 * name that reference (state, command, panel rows); a node-only trigger event picks one of them.
 * A view lookup only; the bindings themselves are the project's. */
export type OriginRole = 'direct' | 'single' | 'open' | 'closed' | 'control' | 'relay' | 'measurement' | 'trigger';
/** `words`: what was sent ("open", "on"); `row`: the point panel row or relay/measurement row. */
export type Origin = { nodeId: string; role: OriginRole; kind: EquipmentKind; label: string; row?: string; words?: string };
/** `at`: when the signal changed; an origin only counts near a trigger event. */
export type OriginLookup = (iedId: string, ref: string, at: number) => Origin | null;

/** State bindings win over a command binding, which wins over panel rows. */
const PRIORITY: Record<OriginRole, number> = { direct: 0, single: 0, open: 0, closed: 0, control: 1, relay: 2, measurement: 2,
  trigger: 0 };

/** Same data, or one is a part of the other ("LD/XCBR1.Pos" and "LD/XCBR1.Pos.stVal"). */
export function refMatches(bound: string, changed: string): boolean {
  return bound === changed || changed.startsWith(`${bound}.`) || bound.startsWith(`${changed}.`);
}

type Entry = { ref: string; origin: Origin; rank: number; y: number; x: number };

function entries(node: EquipmentNode): [string, Entry][] {
  const data = node.data;
  const base = { nodeId: node.id, kind: data.kind, label: data.label };
  const at = (iedId: string, ref: string, role: OriginRole, row?: string): [string, Entry] =>
    [iedId, { ref, origin: { ...base, role, ...(row ? { row } : {}) }, rank: PRIORITY[role], y: node.position.y, x: node.position.x }];
  const result: [string, Entry][] = [];
  const mapping = data.state_mapping;
  if (mapping?.mode === 'contacts_to_dbpos') {
    result.push(at(mapping.open.ied_id, mapping.open.ref, 'open'), at(mapping.closed.ied_id, mapping.closed.ref, 'closed'));
  } else if (mapping) {
    result.push(at(mapping.source.ied_id, mapping.source.ref, mapping.mode === 'direct' ? 'direct' : 'single'));
  }
  if (data.control_binding) result.push(at(data.control_binding.ied_id, data.control_binding.ref, 'control'));
  for (const item of data.relay_statuses || []) result.push(at(item.source.ied_id, item.source.ref, 'relay', item.label));
  for (const item of data.measurements || []) result.push(at(item.source.ied_id, item.source.ref, 'measurement', item.label));
  return result;
}

/** Every symbol whose bindings name the reference, best first (state, then command, then rows;
 * ties to the topmost, then leftmost). Built once per drawing. */
export function originCandidates(nodes: EquipmentNode[]): (iedId: string, ref: string) => Origin[] {
  const byIed = new Map<string, Entry[]>();
  for (const node of nodes) for (const [iedId, entry] of entries(node)) byIed.set(iedId, [...(byIed.get(iedId) || []), entry]);
  return (iedId, ref) => (byIed.get(iedId) || []).filter(entry => refMatches(entry.ref, ref))
    .sort((a, b) => a.rank - b.rank || a.y - b.y || a.x - b.x).map(entry => entry.origin);
}

const SHORT: Partial<Record<EquipmentKind, string>> = { SOURCE: 'Source', RELAY: 'Relay', TRIP_RELAY: 'Trip relay',
  PUSH_BUTTON: 'Button', JUNCTION: 'Junction', INFEED: 'Infeed', MEASUREMENT_LIST: 'Measurements', POINT_PANEL: 'Panel',
  TAP_TRANSFORMER: 'Transformer' };

/** Short symbol name for words: "CB Y130"; a label that already starts with its kind stays as it is. */
export function symbolName(origin: Pick<Origin, 'kind' | 'label'>): string {
  const kind = SHORT[origin.kind] ?? origin.kind;
  const label = origin.label.trim();
  if (!label) return kind;
  return label.toUpperCase().startsWith(kind.toUpperCase()) ? label : `${kind} ${label}`;
}
