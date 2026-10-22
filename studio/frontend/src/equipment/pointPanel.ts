import type { Control, PanelPoint, PointType, Signal, SignalBinding } from '../api/types.ts';
import { MEASUREMENT_DEFAULTS } from './measurementDisplay.ts';
import { shortRef } from './relayDisplay.ts';

/** Point panel rules (ADR-0102); the Go engine checks the same limits. */
export const MAX_PANEL_POINTS = 300;
export const MAX_POINT_LABEL = 100;
export const MAX_POINT_GROUP = 100;
export const POINT_TYPES: readonly PointType[] = ['status', 'double', 'command', 'value'];

/** Signal kinds each type accepts (status = Boolean, double = Dbpos, value = numeric MX). */
export const POINT_SIGNAL_KINDS: Record<Exclude<PointType, 'command'>, string[]> = {
  status: ['BOOLEAN'], double: ['Dbpos'],
  value: ['INT8', 'INT16', 'INT32', 'INT64', 'INT128', 'INT8U', 'INT16U', 'INT24U', 'INT32U', 'FLOAT32', 'FLOAT64'],
};

/** Dbpos positions a double point can hold, in the order the buttons are shown. */
export const DOUBLE_POSITIONS: { raw: number; word: string }[] = [
  { raw: 1, word: 'Open' }, { raw: 2, word: 'Closed' }, { raw: 0, word: 'Intermediate' }, { raw: 3, word: 'Bad' },
];

/** The next free row id: "p1", "p2" … after the highest number in use, so ids never repeat in a panel. */
export function nextPointId(points: readonly PanelPoint[]): string {
  let highest = 0;
  for (const point of points) {
    const found = /^p(\d+)$/.exec(point.id);
    if (found) highest = Math.max(highest, Number(found[1]));
  }
  return `p${highest + 1}`;
}

const VALUE_DEFAULTS = { ...MEASUREMENT_DEFAULTS, pinned: false, healthy: null } as const;

/** A new status, double or value row for a chosen signal: its description or short reference as the name. */
export function newSignalPoint(type: Exclude<PointType, 'command'>, binding: SignalBinding,
  signal: Pick<Signal, 'ref' | 'description'> & { kind?: string }, id: string, iedName?: string): PanelPoint {
  const label = (signal.description.trim() || shortRef(signal.ref, iedName)).slice(0, MAX_POINT_LABEL);
  // A whole number (a tap position, a counter) shows without decimals (change 0173).
  const decimals = signal.kind?.startsWith('INT') ? 0 : VALUE_DEFAULTS.decimals;
  return { id, label, group: '', type, source: binding, control: null, ...VALUE_DEFAULTS, decimals };
}

/** A new command row for a control of the IED. */
export function newCommandPoint(iedId: string, control: Pick<Control, 'ref'>, id: string, iedName?: string): PanelPoint {
  return { id, label: shortRef(control.ref, iedName).slice(0, MAX_POINT_LABEL), group: '', type: 'command',
    source: null, control: { ied_id: iedId, ref: control.ref }, ...VALUE_DEFAULTS };
}

/** The signal or control reference a row is bound to. */
export const pointRef = (point: PanelPoint): string => point.source?.ref ?? point.control?.ref ?? '';

/** Rows that read a signal (status, double, value) have a live value; commands have none. */
export const hasLiveValue = (point: PanelPoint): boolean => point.type !== 'command';

/** "type:ref" keys of the rows, so a picker can mark what the panel already holds. */
export function pointKeys(points: readonly PanelPoint[]): Set<string> {
  return new Set(points.map(point => `${point.type}:${pointRef(point)}`));
}

export type PointPatch = Partial<Pick<PanelPoint, 'label' | 'group' | 'unit' | 'scale' | 'offset' | 'decimals' | 'pinned' | 'healthy'>>;

/** Changes one row; other rows are kept as they are. */
export function patchPoint(points: PanelPoint[], id: string, patch: PointPatch): PanelPoint[] {
  return points.map(point => point.id === id ? { ...point, ...patch } : point);
}

/** Adds rows up to the limit; says how many did not fit. */
export function addPoints(points: PanelPoint[], fresh: PanelPoint[]): { points: PanelPoint[]; skipped: number } {
  const room = Math.max(0, MAX_PANEL_POINTS - points.length);
  return { points: [...points, ...fresh.slice(0, room)], skipped: Math.max(0, fresh.length - room) };
}

/** Gives rows added together their ids, each after the one before (the ids they came with are replaced). */
export function withIds(points: readonly PanelPoint[], fresh: readonly PanelPoint[]): PanelPoint[] {
  const result: PanelPoint[] = [];
  for (const item of fresh) result.push({ ...item, id: nextPointId([...points, ...result]) });
  return result;
}
