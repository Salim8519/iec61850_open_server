import type { EquipmentNode } from '../api/types';
import { originCandidates, refMatches, type OriginLookup } from './signalOrigin.ts';

/** Where a GOOSE change starts on the SLD (ADR-0112): the component that triggered it, and only
 * that. A trigger is a live event of a user action on one component: a GOOSE send (equipment link
 * or IED component), a point panel row, or an operate / test / button press of equipment. A change
 * with no trigger near it (a real IED, a generator, a script, an outside client) starts at the
 * sender's IED component. Pure: the store keeps the triggers, the overlay resolves them. */
export const TRIGGER_MS = 1500;

export type Trigger = { key: number; nodeId: string; at: number; iedId?: string; refs?: string[]; words?: string; row?: string };
type Event = { kind: string; [key: string]: unknown };

const ACTION_WORDS: Record<string, string> = { open: 'open', closed: 'closed', on: 'on', off: 'off' };
/** Events that name only the component; its own bindings say which values it moved. */
const NODE_ONLY = new Set(['equipment_feedback_applied', 'equipment_test', 'equipment_test_command',
  'equipment_button_applied', 'tap_command_requested']);

const text = (value: unknown) => typeof value === 'string' ? value : '';

/** The trigger one live event carries, or null. */
export function triggerOf(event: Event, at: number, key: number): Trigger | null {
  const nodeId = text(event.node_id);
  if (!nodeId) return null;
  if (event.kind === 'goose_send_requested' && text(event.ied_id) && Array.isArray(event.refs)) {
    const words = ACTION_WORDS[text(event.action)];
    return { key, nodeId, at, iedId: text(event.ied_id), refs: event.refs.map(String), ...(words ? { words } : {}) };
  }
  if (event.kind === 'point_panel_requested' && text(event.ied_id) && text(event.ref)) {
    const row = text(event.point_id);
    return { key, nodeId, at, iedId: text(event.ied_id), refs: [text(event.ref)], ...(row ? { row } : {}) };
  }
  return NODE_ONLY.has(event.kind) ? { key, nodeId, at } : null;
}

/** The origin lookup for one drawing and the recent triggers: a trigger naming the changed value
 * (exactly, or the control object of that status) wins; else a node-only trigger whose own
 * bindings name it; else null (the change starts at its sender's IED component). */
export function triggeredOrigins(nodes: EquipmentNode[], triggers: Trigger[], window = TRIGGER_MS): OriginLookup {
  const byId = new Map(nodes.map(node => [node.id, node]));
  const candidates = originCandidates(nodes);
  return (iedId, ref, at) => {
    const near = triggers.filter(item => byId.has(item.nodeId) && Math.abs(item.at - at) <= window)
      .sort((a, b) => Math.abs(a.at - at) - Math.abs(b.at - at));
    for (const item of near) {
      if (!item.refs || item.iedId !== iedId || !item.refs.some(named => refMatches(named, ref))) continue;
      const node = byId.get(item.nodeId)!;
      const row = item.row ? node.data.points?.find(point => point.id === item.row)?.label : undefined;
      return { nodeId: node.id, role: 'trigger', kind: node.data.kind, label: node.data.label,
        ...(row ? { row } : {}), ...(item.words ? { words: item.words } : {}) };
    }
    const acted = new Set(near.filter(item => !item.refs).map(item => item.nodeId));
    return candidates(iedId, ref).find(origin => acted.has(origin.nodeId)) ?? null;
  };
}
