import { api } from './client';

export type ValueStatus = 'ok' | 'warning' | 'problem' | 'error' | 'pending';
export type Attribute = { ref: string; fc: string; path: string; type: string };
export type DataObject = { name: string; fcs: string[]; attributes: Attribute[] };
export type NodeView = { ref: string; name: string; objects: DataObject[]; leaf_count: number;
  datasets: { name: string; ref: string; members: string[] }[];
  reports: { name: string; ref: string; buffered: boolean }[] };
export type ReadValue = { raw: unknown; text: string; status: ValueStatus; at: number | null };
export type WatchItem = { ref: string; fc: string; type: string; text: string; status: ValueStatus; at: number | null };
export type ExplorerStatus = { state: 'disconnected' | 'connecting' | 'connected' | 'failed' | 'lost'; message: string;
  host: string; port: number; watch: WatchItem[]; interval_ms: number; polling: boolean; reads: number };

export const explorerApi = {
  status: () => api<ExplorerStatus>('/tools/explorer'),
  connect: (host: string, port: number) => api<ExplorerStatus>('/tools/explorer/connect', 'POST', { host, port }),
  disconnect: () => api<ExplorerStatus>('/tools/explorer/disconnect', 'POST', {}),
  devices: () => api<{ names: string[] }>('/tools/explorer/devices'),
  nodes: (device: string) => api<{ names: string[] }>(`/tools/explorer/nodes?device=${encodeURIComponent(device)}`),
  node: (ref: string) => api<NodeView>(`/tools/explorer/node?ref=${encodeURIComponent(ref)}`),
  read: (items: { ref: string; fc: string }[]) =>
    api<{ values: Record<string, ReadValue> }>('/tools/explorer/read', 'POST', { items }),
  watch: (items: { ref: string; fc: string; type?: string }[], interval_ms?: number) =>
    api<ExplorerStatus>('/tools/explorer/watch', 'PUT', { items, interval_ms }),
  polling: (on: boolean) => api<ExplorerStatus>('/tools/explorer/polling', 'POST', { on }),
  csvUrl: '/api/v1/tools/explorer/watch.csv',
};

export type FoundDevice = { ip: string; port: number; tcp_ms: number; identified: boolean | null; vendor: string;
  model: string; revision: string; devices: string[]; reason: string; connect_ms?: number };
export type ScanStatus = { state: 'idle' | 'scanning' | 'done' | 'stopped' | 'failed'; range: string; port: number;
  total: number; probed: number; found: FoundDevice[]; message: string; seconds: number };
export type ScanPreview = { range: string; count: number; first: string; last: string; port: number };

export const scanApi = {
  status: () => api<ScanStatus>('/tools/scan'),
  preview: (range: string, port: number) => api<ScanPreview>('/tools/scan/preview', 'POST', { range, port }),
  start: (range: string, port: number) => api<ScanStatus>('/tools/scan/start', 'POST', { range, port }),
  stop: () => api<ScanStatus>('/tools/scan/stop', 'POST', {}),
};
