import { api } from './client';

export type Role = 'gateway' | 'master' | 'converter';
export type MasterProtocol = 'iec104' | 'dnp3' | 'modbus';
export type CommandKind = 'single' | 'double' | 'setpoint_float' | 'setpoint_scaled' | 'setpoint_normalized' | 'clock'
  | 'binary_output' | 'analog_output_int' | 'analog_output_float' | 'coil' | 'register' | 'registers_int32' | 'registers_float';
export type CrobCode = 'latch' | 'pulse' | 'trip_close';
export type CommandRoute = { upstream: number; upstream_kind: string; kind: string; address: number; select: boolean };
export type ModbusRange = { table: string; start: number; count: number; data_type: string; word_order: string };
export type MasterConfig = { role: Role; protocol: MasterProtocol; host: string; auto_connect: boolean;
  settings: Record<MasterProtocol, Record<string, number | boolean>>; modbus_ranges: ModbusRange[]; routes: CommandRoute[] };
export type MasterStatus = { state: 'disconnected' | 'connected' | 'lost' | 'failed'; message: string; link_open: boolean;
  protocol: MasterProtocol; host: string; port: number; points: number; last_interrogation: number | null;
  available: boolean; traffic_last: number; auto_connect: boolean; probe: ProbeState;
  poll_failures: { range: number; text: string }[]; ranges: number };
export type ProbeState = { state: 'idle' | 'running' | 'done' | 'failed'; table?: string; start?: number; end?: number;
  requests?: number; address?: number; found?: { table: string; start: number; count: number }[]; message?: string };
export type MasterPoint = { key: string; group: string; group_name: string; type: string; address: number; value: unknown;
  value_text: string; quality_text: string; good: boolean; time_ms: number | null; cause: string; changes: number;
  label: string; ticked: boolean; stale: boolean; command: CommandKind | null };
export type MasterCommand = { kind: CommandKind; address?: number; value?: number; select?: boolean; code?: CrobCode };
export type CommandAnswer = { ok: boolean; text: string; terminated?: boolean };
export type MasterCommandEntry = { seq: number; at: number; who: string; kind: CommandKind; address: number | null;
  value: number | null; select: boolean; state: 'sent' | 'confirmed' | 'refused'; text: string };
type Page<T> = { entries: T[]; last: number; dropped: number };

/** Paths the pages poll through the shared live feed (speed lane F). */
export const MASTER_STATUS = '/tools/master';
export const MASTER_CONFIG = '/tools/master/config';
export const MASTER_POINTS = '/tools/master/points';
export const MASTER_COMMANDS = '/tools/master/commands';

export const masterApi = {
  setRole: (role: Role) => api<{ role: Role }>('/tools/gateway/role', 'POST', { role }),
  status: () => api<MasterStatus>('/tools/master'),
  config: () => api<MasterConfig>('/tools/master/config'),
  saveConfig: (protocol: MasterProtocol, host: string, settings: Record<string, number | boolean>) =>
    api<MasterConfig>('/tools/master/config', 'PUT', { protocol, host, settings }),
  connect: () => api<MasterStatus>('/tools/master/connect', 'POST', {}),
  disconnect: () => api<MasterStatus>('/tools/master/disconnect', 'POST', {}),
  interrogate: () => api<{ confirmed: boolean; text: string }>('/tools/master/interrogate', 'POST', {}),
  points: () => api<{ points: MasterPoint[] }>('/tools/master/points'),
  untick: (unticked: string[]) => api<{ unticked: string[] }>('/tools/master/selection', 'PUT', { unticked }),
  label: (key: string, label: string) => api<{ key: string; label: string }>('/tools/master/label', 'PUT', { key, label }),
  csvUrl: () => '/api/v1/tools/master/points.csv',
  importLabels: (text: string) => api<{ imported: number; problems: string[] }>('/tools/master/labels.csv', 'POST', { text }),
  command: (command: MasterCommand) => api<CommandAnswer>('/tools/master/command', 'POST', { ...command, confirm: true }),
  saveRanges: (modbus_ranges: ModbusRange[]) => api<{ modbus_ranges: ModbusRange[] }>('/tools/master/ranges', 'PUT', { modbus_ranges }),
  saveRoutes: (routes: CommandRoute[]) => api<{ routes: CommandRoute[] }>('/tools/master/routes', 'PUT', { routes }),
  probe: (table: string, start: number, end: number) => api<ProbeState>('/tools/master/probe', 'POST', { table, start, end }),
  addFound: (data_type: string) => api<{ modbus_ranges: ModbusRange[] }>('/tools/master/probe/ranges', 'POST', { data_type }),
  traffic: (after: number) => api<Page<{ seq: number; at: number; dir: string; peer: string; text: string }>>(
    `/tools/master/traffic?after=${after}`),
  commands: () => api<Page<MasterCommandEntry>>('/tools/master/commands'),
};
