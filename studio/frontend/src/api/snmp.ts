import { api } from './client';

/** SNMP devices tool (ADR-0092, Go engine only). Values are text: numbers in decimal, OIDs dotted. */
export type SnmpRole = 'simulate' | 'read';
export type PointType = 'integer' | 'octets' | 'oid' | 'ipaddress' | 'counter32' | 'gauge32' | 'timeticks'
  | 'counter64' | 'opaque' | 'null';
export type Behaviour = 'fixed' | 'uptime' | 'counter';
export type Level = 'noAuthNoPriv' | 'authNoPriv' | 'authPriv';
export type SnmpVersion = 'v1' | 'v2c' | 'v3';
export type SnmpPoint = { oid: string; type: PointType; value: string; hex?: boolean; behaviour?: Behaviour;
  rate?: number; writable?: boolean };
export type SnmpUser = { name: string; level: Level; auth?: string; auth_password?: string; priv?: string;
  priv_password?: string };
export type TrapTarget = { host: string; port: number; version: SnmpVersion; community?: string; user?: string;
  inform?: boolean };
export type SnmpDevice = { id: string; name: string; profile: string; ip: string; port: number; v1: boolean;
  v2c: boolean; v3: boolean; read_community: string; write_community: string; users: SnmpUser[]; engine_id: string;
  traps: TrapTarget[]; points: SnmpPoint[] };
export type SnmpTarget = { host: string; port: number; version: SnmpVersion; community?: string; user: SnmpUser;
  timeout_ms?: number; retries?: number };
export type SnmpConfig = { role: SnmpRole; auto_start: boolean; devices: SnmpDevice[]; reader: {
  target: SnmpTarget; every_ms: number; watched: string[]; trap_ip: string; trap_port: number;
  trap_users: SnmpUser[]; traps_on: boolean; engine_id: string } };
export type DeviceStatus = { id: string; name: string; profile: string; kind: string; address: string;
  state: 'running' | 'stopped' | 'failed'; error: string; points: number };
export type ReaderState = 'disconnected' | 'connecting' | 'walking' | 'connected' | 'failed';
export type SnmpStatus = { role: SnmpRole; engine: 'off' | 'running' | 'failed'; message: string;
  devices: DeviceStatus[]; running: number;
  reader: { state: ReaderState; message: string; target: string; system: ValueRow[]; engine_id: string;
    walk_count: number; values: number; polled: number; every_ms: number; problem: string };
  traps: { listening: boolean; address: string; message: string }; requests_last: number; traps_last: number };
/** A value as the page shows it: MIB name, what it means ("up", "3 days, 4:05:06.07") and access. */
export type ValueRow = SnmpPoint & { name: string; meaning: string; writable: boolean; key?: boolean; polled?: boolean };
export type RequestEntry = { seq: number; at: number; device: string; peer: string; version: string; op: string;
  count: number; oid: string; name: string; result: string; held: number };
export type TrapEntry = { seq: number; at: number; from: string; version: string; kind: string; community?: string;
  user?: string; notification?: string; notification_name: string; uptime?: number; problem?: string;
  varbinds: ValueRow[] };
export type MibModule = { name: string; file: string; builtin: boolean; nodes: number; missing: string[];
  problems: string[] };
export type Action = { id: string; label: string };
export type TrapResult = { target: string; ok: boolean; text: string };
export type Imported = { device: SnmpDevice; problems: string[] };
type Page<T> = { entries: T[]; last: number; dropped: number };

/** Paths the page reads through the shared live feed (push on the Go engine). */
export const SNMP_STATUS = '/tools/snmp';
export const SNMP_VALUES = '/tools/snmp/read/values';
export const SNMP_REQUESTS = '/tools/snmp/requests';
export const SNMP_TRAPS = '/tools/snmp/traps';
export const snmpPointsPath = (id: string) => `/tools/snmp/devices/${encodeURIComponent(id)}/points`;

const device = (id: string) => `/tools/snmp/devices/${encodeURIComponent(id)}`;

export const snmpApi = {
  status: () => api<SnmpStatus>(SNMP_STATUS),
  config: () => api<SnmpConfig>('/tools/snmp/config'),
  setRole: (role: SnmpRole) => api<SnmpStatus>('/tools/snmp/role', 'POST', { role }),
  start: () => api<SnmpStatus>('/tools/snmp/start', 'POST', {}),
  stop: () => api<SnmpStatus>('/tools/snmp/stop', 'POST', {}),
  add: (profile: string) => api<{ device: SnmpDevice }>('/tools/snmp/devices', 'POST', { profile }),
  update: (value: SnmpDevice) => api<{ device: SnmpDevice }>(device(value.id), 'PUT', { device: value }),
  remove: (id: string) => api<object>(device(id), 'DELETE'),
  points: (id: string) => api<{ points: ValueRow[]; live: boolean }>(snmpPointsPath(id)),
  savePoints: (id: string, points: SnmpPoint[]) => api<object>(`${device(id)}/points`, 'PUT', { points }),
  setValue: (id: string, oid: string, value: string, hex = false) =>
    api<{ point: SnmpPoint }>(`${device(id)}/value`, 'PUT', { oid, value, hex }),
  actions: (id: string) => api<{ actions: Action[] }>(`${device(id)}/actions`),
  trap: (id: string, action: string) => api<{ results: TrapResult[] }>(`${device(id)}/trap`, 'POST', { action }),
  csvUrl: (id: string) => `/api/v1${device(id)}/points.csv`,
  importDevice: (kind: 'walk' | 'csv', name: string, text: string) =>
    api<Imported>('/tools/snmp/import', 'POST', { kind, name, text }),
  connect: (target: SnmpTarget) => api<SnmpStatus>('/tools/snmp/read/connect', 'POST', { target }),
  disconnect: () => api<SnmpStatus>('/tools/snmp/read/disconnect', 'POST', {}),
  walk: (root = '') => api<SnmpStatus>('/tools/snmp/read/walk', 'POST', { root }),
  setPoll: (every_ms: number) => api<SnmpStatus>('/tools/snmp/read/poll', 'PUT', { every_ms }),
  watch: (oids: string[]) => api<SnmpStatus>('/tools/snmp/read/watch', 'PUT', { oids }),
  write: (point: SnmpPoint) => api<{ values: ValueRow[] }>('/tools/snmp/read/set', 'POST', { point, confirm: true }),
  saveTwin: (name: string) => api<Imported>('/tools/snmp/read/save', 'POST', { name }),
  listen: (ip: string, port: number, users: SnmpUser[]) =>
    api<SnmpStatus>('/tools/snmp/traps/listen', 'POST', { ip, port, users }),
  stopListening: () => api<SnmpStatus>('/tools/snmp/traps/stop', 'POST', {}),
  requests: (after: number) => api<Page<RequestEntry>>(`${SNMP_REQUESTS}?after=${after}`),
  traps: (after: number) => api<Page<TrapEntry>>(`${SNMP_TRAPS}?after=${after}`),
  mibs: () => api<{ modules: MibModule[] }>('/tools/snmp/mibs'),
  importMib: (name: string, text: string) =>
    api<{ imported: string[]; modules: MibModule[] }>('/tools/snmp/mibs', 'POST', { name, text }),
  removeMib: (module: string) => api<{ modules: MibModule[] }>(`/tools/snmp/mibs/${encodeURIComponent(module)}`, 'DELETE'),
  restoreMibs: () => api<{ modules: MibModule[] }>('/tools/snmp/mibs/restore', 'POST', {}),
  resolve: (name: string) => api<{ oid: string; name: string }>(`/tools/snmp/mibs/resolve?name=${encodeURIComponent(name)}`),
};
