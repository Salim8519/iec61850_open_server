// The SCADA master's own point list (ADR-0110, Go engine only): addresses and types the master
// expects, compared by the engine with what the device sends. Planning routes never save; the page
// saves the whole list with `save` (save always, ADR-0101), which the engine checks again.
import { api } from './client';
import type { CommandKind, CrobCode, MasterCommand, MasterProtocol, ModbusRange } from './master';

/** One choice of the master's type list (the engine's catalog, other direction of the gateway's). */
export type MasterType = { kind: string; name: string; type_id: string; most_used: boolean; group: string; command: string;
  table?: string; data_type?: string; registers?: number };
export type CopiedFrom = { protocol: string; address: number; kind: string };
export type ListPoint = { id: string; name: string; type: string; address: number; scale: number; offset: number; unit: string;
  table?: string; data_type?: string; registers?: number; word_order?: string;
  command?: CommandKind; select?: boolean; value?: number; code?: CrobCode; from?: CopiedFrom };
export type MasterList = { protocol: MasterProtocol; points: ListPoint[]; types: MasterType[]; check_ied: boolean;
  auto_ranges: ModbusRange[] };
export type ListRowCells = Record<string, string>;
export type ListPlanRow = { number: number; point: ListPoint; problem: string; cells?: ListRowCells };
/** `points` holds the new points only when every row can be added (`ok`). */
export type ListPlan = { rows: ListPlanRow[]; points: ListPoint[]; ok: boolean; columns?: string[]; notes?: string[];
  gateway_points?: number };
export type PointState = 'received' | 'missing' | 'wrong_type' | 'waiting' | 'command';
export type IedCheck = { ok: boolean | null; text: string; source: string };
export type ComparedRow = { id: string; point: ListPoint; state: PointState; reason: string; command: MasterCommand | null;
  received_key?: string; received_group?: string; received_type?: string; value?: unknown; value_text?: string;
  engineering?: string; quality_text?: string; good?: boolean; stale?: boolean; time_ms?: number | null; cause?: string;
  check?: IedCheck };
export type ExtraRow = { key: string; group: string; group_name: string; address: number; type: string; label: string;
  value_text: string; quality_text: string; good: boolean; list_type: string | null };
export type Counts = Record<PointState | 'not_in_list' | 'total', number>;
export type Comparison = { rows: ComparedRow[]; not_in_list: ExtraRow[]; counts: Counts; asked: boolean; connected: boolean;
  check_ied: boolean };

export const MASTER_COMPARE = '/tools/master/list/compare';

export const masterListApi = {
  list: () => api<MasterList>('/tools/master/list'),
  save: (points: ListPoint[]) => api<MasterList>('/tools/master/list', 'PUT', { points }),
  checkIed: (on: boolean) => api<{ check_ied: boolean }>('/tools/master/list/check-ied', 'PUT', { on }),
  one: (point: Partial<ListPoint>) => api<ListPlan>('/tools/master/list/new', 'POST', point),
  range: (type: string, count: number, from: number, name: string) =>
    api<ListPlan>('/tools/master/list/new', 'POST', { type, count, from, name }),
  rows: (text: string) => api<ListPlan>('/tools/master/list/rows', 'POST', { text }),
  uploaded: (data: string) => api<ListPlan>('/tools/master/list.xlsx', 'POST', { data }),
  templateUrl: () => '/api/v1/tools/master/list.xlsx',
  csvUrl: () => '/api/v1/tools/master/list.csv',
  copyGateway: () => api<ListPlan>('/tools/master/list/copy-gateway', 'POST', {}),
  fromReceived: (keys: string[]) => api<ListPlan>('/tools/master/list/from-received', 'POST', { keys }),
  compare: () => api<Comparison>(MASTER_COMPARE),
};
