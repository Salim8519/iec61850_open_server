import { api } from './client';

/** What runs outside the IED Run right now (GET /activity); the page words it. */
export type Activity = {
  gateway: { state: 'stopped' | 'running' | 'failed'; protocol: string; port: number | null; clients: number };
  explorer: { state: string; host: string; polling: boolean; watched: number };
  scan: { state: string; probed: number; total: number };
  master: { state: 'disconnected' | 'connected' | 'lost' | 'failed'; protocol: string; host: string; points: number };
  sequences: number;
  /** Go engine only, read from GET /tools/snmp (not part of /activity). */
  snmp?: { running: number; reader: string; target: string; listening: boolean };
};

export const activityApi = { get: () => api<Activity>('/activity') };
