import { api } from './client';

export type Protocol = 'iec104' | 'modbus' | 'dnp3';
export type Listen = 'this_pc' | 'network';
export type PointKind = 'single' | 'double' | 'float' | 'integer' | 'command_single' | 'command_double';
export type GatewayPoint = { id?: string; kind: PointKind; address: number; ied: string; ref: string; scale: number;
  description: string; ied_name?: string; monitor_only?: boolean; value_text?: string; online?: boolean };
export type GatewayConfig = { protocol: Protocol; listen: Listen; settings: Record<Protocol, Record<string, number | boolean>>;
  real_commands: boolean; auto_start: boolean };
export type GatewayStatus = { state: 'stopped' | 'running' | 'failed'; message: string; protocol: Protocol; listen: Listen;
  opened: Record<string, string | number | boolean> | null; clients: string[]; problems: string[]; run_state: string;
  available: boolean; traffic_last: number; real_commands: boolean; auto_start: boolean };
export type TrafficEntry = { seq: number; at: number; dir: string; peer: string; text: string };
export type CommandEntry = { seq: number; at: number; label: string; protocol: string; peer: string; request: string;
  applied: string; observed: string };
type Page<T> = { entries: T[]; last: number; dropped: number };

/** The per-point problems the backend lists when a point list cannot be saved. */
export const problemsOf = (error: unknown): string[] =>
  ((error as { data?: { problems?: string[] } }).data?.problems) ?? [];

export const gatewayApi = {
  status: () => api<GatewayStatus>('/tools/gateway'),
  config: () => api<GatewayConfig>('/tools/gateway/config'),
  saveConfig: (protocol: Protocol, listen: Listen, settings: Record<string, number | boolean>, real_commands: boolean) =>
    api<GatewayConfig>('/tools/gateway/config', 'PUT', { protocol, listen, settings, real_commands }),
  points: (protocol: Protocol) => api<{ points: GatewayPoint[] }>(`/tools/gateway/points?protocol=${protocol}`),
  savePoints: (protocol: Protocol, points: GatewayPoint[]) =>
    api<{ points: GatewayPoint[] }>('/tools/gateway/points', 'PUT', { protocol, points }),
  propose: (protocol: Protocol) => api<{ points: GatewayPoint[]; notes: string[] }>('/tools/gateway/propose', 'POST', { protocol }),
  importCsv: (protocol: Protocol, text: string) =>
    api<{ points: GatewayPoint[]; problems: string[] }>('/tools/gateway/points.csv', 'POST', { protocol, text }),
  csvUrl: (protocol: Protocol) => `/api/v1/tools/gateway/points.csv?protocol=${protocol}`,
  start: () => api<GatewayStatus>('/tools/gateway/start', 'POST', {}),
  stop: () => api<GatewayStatus>('/tools/gateway/stop', 'POST', {}),
  traffic: (after: number) => api<Page<TrafficEntry>>(`/tools/gateway/traffic?after=${after}`),
  commands: () => api<Page<CommandEntry>>('/tools/gateway/commands'),
};
