import type { Diagram, EquipmentNode, PanelPoint } from '../api/types.ts';
import { MEASUREMENT_DEFAULTS } from './measurementDisplay.ts';
import { MAX_PANEL_POINTS, MAX_POINT_LABEL } from './pointPanel.ts';

/** Push buttons into point panels (change 0180): one panel per IED, placed where its buttons were.
 *  A latching status button becomes an ON/OFF status row (the row's ON/OFF sets the same signal in the
 *  simulated IED); a command button becomes a command row. Buttons with wires, self-resetting buttons
 *  and buttons with no IED binding are left as they are. One diagram change, so one Undo. */
export const PANEL_GROUPS = { status: 'Status inputs', command: 'Commands' } as const;

export type PushConversion = { diagram: Diagram; panels: string[]; converted: number; kept: number };

const ROW_DEFAULTS = { ...MEASUREMENT_DEFAULTS, pinned: false, healthy: null } as const;

type Row = { iedId: string; button: EquipmentNode; point: Omit<PanelPoint, 'id'> };

function rowOf(button: EquipmentNode, wired: Set<string>): Row | null {
  const data = button.data;
  if (data.kind !== 'PUSH_BUTTON' || wired.has(button.id)) return null;
  const label = (data.label || '').trim().slice(0, MAX_POINT_LABEL) || 'Button';
  const mode = data.push_mode ?? 'latching_status';
  const source = data.state_mapping?.mode === 'direct' ? data.state_mapping.source : null;
  if (mode === 'latching_status' && source?.ied_id) {
    return { iedId: source.ied_id, button, point: { label, group: PANEL_GROUPS.status, type: 'status',
      source, control: null, ...ROW_DEFAULTS } };
  }
  const control = data.control_binding;
  if (mode === 'spc_momentary' && control?.ied_id) {
    return { iedId: control.ied_id, button, point: { label, group: PANEL_GROUPS.command, type: 'command',
      source: null, control, ...ROW_DEFAULTS } };
  }
  return null;
}

/** Every unwired latching or command push button that can become a panel row. */
export function convertiblePushButtons(diagram: Diagram): number {
  const wired = wiredIds(diagram);
  return diagram.nodes.filter(node => rowOf(node, wired)).length;
}

function wiredIds(diagram: Diagram): Set<string> {
  return new Set(diagram.edges.flatMap(edge => [edge.source, edge.target]));
}

export function convertPushButtons(diagram: Diagram, iedNames: Record<string, string>): PushConversion {
  const wired = wiredIds(diagram);
  const byIed = new Map<string, Row[]>();
  for (const node of diagram.nodes) {
    const row = rowOf(node, wired);
    if (row) byIed.set(row.iedId, [...(byIed.get(row.iedId) ?? []), row]);
  }
  const removed = new Set<string>();
  const panels: EquipmentNode[] = [];
  for (const [iedId, rows] of byIed) {
    // Status inputs first, then commands, each in the order the buttons were drawn.
    const ordered = [...rows.filter(row => row.point.type === 'status'), ...rows.filter(row => row.point.type === 'command')];
    const left = Math.min(...rows.map(row => row.button.position.x));
    const top = Math.min(...rows.map(row => row.button.position.y));
    const name = iedNames[iedId] || 'IED';
    for (let start = 0, part = 1; start < ordered.length; start += MAX_PANEL_POINTS, part++) {
      const chunk = ordered.slice(start, start + MAX_PANEL_POINTS);
      const points = chunk.map((row, index) => ({ id: `p${index + 1}`, ...row.point }));
      chunk.forEach(row => removed.add(row.button.id));
      panels.push({ id: crypto.randomUUID(), type: 'equipment', position: { x: left + (part - 1) * 260, y: top },
        data: { kind: 'POINT_PANEL', rotation: 0, ied_id: iedId, points,
          label: `${name} points${part > 1 ? ` ${part}` : ''}`.slice(0, MAX_POINT_LABEL) } });
    }
  }
  const pushTotal = diagram.nodes.filter(node => node.data.kind === 'PUSH_BUTTON').length;
  return { diagram: { ...diagram, nodes: [...diagram.nodes.filter(node => !removed.has(node.id)), ...panels] },
    panels: panels.map(panel => panel.id), converted: removed.size, kept: pushTotal - removed.size };
}
