import { api } from './client';

export type Protocol = 'iec104' | 'modbus' | 'dnp3';
export type Listen = 'this_pc' | 'network';
export type PointKind = 'single' | 'double' | 'float' | 'integer' | 'command_single' | 'command_double';
export type GatewayPoint = { id?: string; kind: PointKind; address: number; ied: string; ref: string; scale: number;
  description: string; ied_name?: string; ied_role?: string; monitor_only?: boolean; value_text?: string; online?: boolean;
  /** Own points (lane G): 'manual' = made up by the user, served with `value`; a command may drive `target`. */
  source?: string; value?: number | boolean; target?: string;
  /** Page only: an own point added here and not saved yet. */
  draft?: boolean };
export type GatewayConfig = { protocol: Protocol; listen: Listen; settings: Record<Protocol, Record<string, number | boolean>>;
  real_commands: boolean; auto_start: boolean };
export type GatewayStatus = { state: 'stopped' | 'running' | 'failed'; message: string; protocol: Protocol; listen: Listen;
  opened: Record<string, string | number | boolean> | null; clients: string[]; problems: string[]; run_state: string;
  available: boolean; traffic_last: number; real_commands: boolean; auto_start: boolean;
  role?: 'gateway' | 'master' | 'converter' };
export type TrafficEntry = { seq: number; at: number; dir: string; peer: string; text: string };
export type CommandEntry = { seq: number; at: number; label: string; protocol: string; peer: string; request: string;
  applied: string; observed: string; point_id?: string | null };
type Page<T> = { entries: T[]; last: number; dropped: number };

/** The per-point problems the backend lists when a point list cannot be saved. */
export const problemsOf = (error: unknown): string[] =>
  ((error as { data?: { problems?: string[] } }).data?.problems) ?? [];

/** Paths the pages poll through the shared live feed (speed lane F). */
export const GATEWAY_STATUS = '/tools/gateway';
export const GATEWAY_CONFIG = '/tools/gateway/config';
export const GATEWAY_COMMANDS = '/tools/gateway/commands';
export const gatewayPointsPath = (protocol: Protocol) => `/tools/gateway/points?protocol=${protocol}`;

export const gatewayApi = {
  status: () => api<GatewayStatus>('/tools/gateway'),
  config: () => api<GatewayConfig>('/tools/gateway/config'),
  saveConfig: (protocol: Protocol, listen: Listen, settings: Record<string, number | boolean>, real_commands: boolean) =>
    api<GatewayConfig>('/tools/gateway/config', 'PUT', { protocol, listen, settings, real_commands }),
  points: (protocol: Protocol) => api<{ points: GatewayPoint[] }>(`/tools/gateway/points?protocol=${protocol}`),
  savePoints: (protocol: Protocol, points: GatewayPoint[]) =>
    api<{ points: GatewayPoint[] }>('/tools/gateway/points', 'PUT', { protocol, points }),
  /** Proposes IED points; `keep` (own points) stays in the list, moved when a proposed point takes its address. */
  propose: (protocol: Protocol, keep: GatewayPoint[] = []) =>
    api<{ points: GatewayPoint[]; notes: string[] }>('/tools/gateway/propose', 'POST', { protocol, keep }),
  /** A new own point for the draft list: the next own id and the next free address of its type. */
  newOwnPoint: (protocol: Protocol, kind: PointKind, points: GatewayPoint[]) =>
    api<{ point: GatewayPoint }>('/tools/gateway/points/new', 'POST', { protocol, kind, points }),
  /** The value of a saved own point; a running gateway sends it to SCADA at once. */
  setOwnValue: (point_id: string, value: number | boolean) =>
    api<{ point: GatewayPoint }>('/tools/gateway/points/value', 'PUT', { point_id, value }),
  importCsv: (protocol: Protocol, text: string) =>
    api<{ points: GatewayPoint[]; problems: string[] }>('/tools/gateway/points.csv', 'POST', { protocol, text }),
  csvUrl: (protocol: Protocol) => `/api/v1/tools/gateway/points.csv?protocol=${protocol}`,
  start: () => api<GatewayStatus>('/tools/gateway/start', 'POST', {}),
  send: (point_id: string, value: boolean) =>
    api<{ ok: boolean; reason: string }>('/tools/gateway/command', 'POST', { point_id, value, confirm: true }),
  stop: () => api<GatewayStatus>('/tools/gateway/stop', 'POST', {}),
  traffic: (after: number) => api<Page<TrafficEntry>>(`/tools/gateway/traffic?after=${after}`),
  commands: () => api<Page<CommandEntry>>('/tools/gateway/commands'),
};
