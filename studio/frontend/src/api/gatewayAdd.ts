// "Add points" of the gateway (ADR-0105, Go engine only). These routes plan and check; the list
// changes only when the page saves it (gatewayApi.savePoints checks it again).
import { api } from './client';
import type { GatewayPoint, PointKind, Protocol } from './gateway';

/** One choice of the type list (the engine's one catalog per protocol). */
export type TypeEntry = { kind: PointKind; name: string; type_id: string; most_used: boolean };
export type IedRef = { id: string; name: string; role: string };
/** A signal or controllable object of one IED that fits the chosen type. */
export type Candidate = { ref: string; name: string; description: string; value_kind: string; fc: string;
  cdc?: string; status_ref?: string; taken: boolean; suggested: boolean };
export type RowCells = { name: string; type: string; ied: string; point: string; address: string; scale: string };
/** One planned row: the point as it would be added, and why it cannot be when `problem` is set. */
export type PlannedRow = { number: number; point: GatewayPoint; problem: string; status_for?: string; cells?: RowCells };
/** `points` holds the new points only when every row can be added (`ok`). */
export type Plan = { rows: PlannedRow[]; points: GatewayPoint[]; ok: boolean; columns?: string[]; ied?: IedRef };
export type Pick = { ref: string; name?: string; address?: number };
export type IedSummary = { id: string; name: string; role: string; offered: number; new: number; ticked: boolean };
export type SmartAnswer = { points: GatewayPoint[]; notes: string[]; summary?: IedSummary[] };

export const addApi = {
  types: (protocol: Protocol) => api<{ types: TypeEntry[] }>(`/tools/gateway/types?protocol=${protocol}`),
  candidates: (protocol: Protocol, ied: string, kind: PointKind, q: string, points: GatewayPoint[]) =>
    api<{ items: Candidate[]; total: number; ied: IedRef }>('/tools/gateway/candidates', 'POST', { protocol, ied, kind, q, points }),
  fromIed: (protocol: Protocol, ied: string, kind: PointKind, picks: Pick[], with_status: boolean, points: GatewayPoint[]) =>
    api<Plan>('/tools/gateway/points/from-ied', 'POST', { protocol, ied, kind, picks, with_status, points }),
  range: (protocol: Protocol, kind: PointKind, count: number, from: number, name: string, points: GatewayPoint[]) =>
    api<Plan>('/tools/gateway/points/new', 'POST', { protocol, kind, count, from, name, points }),
  pasted: (protocol: Protocol, text: string, points: GatewayPoint[]) =>
    api<Plan>('/tools/gateway/points/rows', 'POST', { protocol, text, points }),
  uploaded: (protocol: Protocol, data: string, points: GatewayPoint[]) =>
    api<Plan>('/tools/gateway/points.xlsx', 'POST', { protocol, data, points }),
  templateUrl: (protocol: Protocol) => `/api/v1/tools/gateway/points.xlsx?protocol=${protocol}`,
  /** Smart Propose: `add_new` keeps `current` and adds only missing points; `ieds` are the ticked IEDs. */
  smart: (protocol: Protocol, mode: 'replace' | 'add_new', current: GatewayPoint[], ieds?: string[]) =>
    api<SmartAnswer>('/tools/gateway/propose', 'POST', { protocol, mode, current, ...(ieds ? { ieds } : {}) }),
};

/** A picked file as base64 (the engine reads the workbook). */
export async function fileBase64(file: File): Promise<string> {
  const bytes = new Uint8Array(await file.arrayBuffer());
  let text = '';
  for (let at = 0; at < bytes.length; at += 0x8000) text += String.fromCharCode(...bytes.subarray(at, at + 0x8000));
  return btoa(text);
}
