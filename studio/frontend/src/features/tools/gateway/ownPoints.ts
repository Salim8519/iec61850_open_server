// Own points (lane G): display helpers only. The backend checks every value and address.
import type { GatewayPoint, PointKind } from '../../../api/gateway';
import { OWN_COPY } from '../../../copy/gateway.ts';

export const isOwn = (row: GatewayPoint) => row.source === 'manual';
export const isCommand = (kind: PointKind) => kind.startsWith('command');
export const isMeasured = (kind: PointKind) => kind === 'float' || kind === 'integer' || kind === 'step_position';
const isWhole = (kind: PointKind) => kind === 'integer' || kind === 'step_position';

/** What a person calls an own point: its name, else its id. */
export const ownName = (row: GatewayPoint) => row.description.trim() || row.id || OWN_COPY.newPoint;

export type Choice = { value: string; label: string };

/** The values a status can take, as plain words (the stored number is the value). */
export function statusChoices(kind: PointKind): Choice[] {
  return kind === 'double'
    ? [{ value: '1', label: 'Open (OFF)' }, { value: '2', label: 'Closed (ON)' },
      { value: '0', label: 'Intermediate' }, { value: '3', label: 'Bad state' }]
    : [{ value: '0', label: 'OFF' }, { value: '1', label: 'ON' }];
}

/** The stored number of a status value (true/false from older lists count as ON/OFF). */
export function statusNumber(kind: PointKind, value: GatewayPoint['value']): number {
  if (typeof value === 'boolean') return kind === 'double' ? (value ? 2 : 1) : (value ? 1 : 0);
  if (typeof value === 'number') return value;
  return kind === 'double' ? 1 : 0;
}

/** A typed number for a measured own point, or the plain reason it cannot be used. */
export function parseOwnNumber(text: string, kind: PointKind): { value: number } | { error: string } {
  const trimmed = text.trim().replace(',', '.');
  const value = Number(trimmed);
  if (trimmed === '' || !Number.isFinite(value)) return { error: isWhole(kind) ? OWN_COPY.badWhole : OWN_COPY.badNumber };
  if (kind === 'integer' && (!Number.isInteger(value) || Math.abs(value) > 2147483647)) return { error: OWN_COPY.badWhole };
  if (kind === 'step_position' && (!Number.isInteger(value) || value < -64 || value > 63)) return { error: OWN_COPY.badTap };
  return { value };
}

/** The own points each own command may drive (the backend's targetKinds, ADR-0107). */
const DRIVES: Partial<Record<PointKind, PointKind[]>> = {
  command_step: ['step_position', 'integer'], command_setpoint: ['float', 'integer', 'step_position'],
};

/** The own points a command can drive (single or double for ON/OFF commands), with an id. */
export function ownTargets(rows: GatewayPoint[], kind?: PointKind): Choice[] {
  const fits = (kind && DRIVES[kind]) ?? ['single', 'double'];
  return rows.filter(row => isOwn(row) && row.id && fits.includes(row.kind))
    .map(row => ({ value: row.id!, label: `${ownName(row)} · ${row.address}` }));
}

/** The name of the status point a command drives, or '' when it drives none. */
export function targetName(rows: GatewayPoint[], target: string | undefined): string {
  if (!target) return '';
  const found = rows.find(row => row.id === target);
  return found ? ownName(found) : target;
}

/** Whether a value change goes straight to the backend (saved point, list not edited). */
export const liveEdit = (row: GatewayPoint, dirty: boolean) => Boolean(row.id) && !row.draft && !dirty;
