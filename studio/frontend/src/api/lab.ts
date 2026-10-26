// Response shapes of the "This PC lab network" routes (ADR-0120, Go engine only).
export type LabRow = { ied_id: string; name: string; scl_ip: string; ip: string; yours: boolean; on_card: boolean;
  status: 'ok' | 'same_range' | 'real_device' | 'checking' | 'duplicate' | 'other_card' | 'invalid' | 'missing';
  hint: string; card?: string; alternative?: string };
export type LabRealCard = { name: string; kind: string; connected: boolean; networks: string[] };
export type LabStatus = { installed: boolean; running: boolean; card: string; card_present: boolean; card_name: string;
  ready: boolean; active: boolean; ieds_running: boolean; real_cards: LabRealCard[]; rows: LabRow[];
  checking: boolean; problem?: string; card_addresses: string[] };
export type LabPing = { ok: boolean; ms: number; ttl: number };
export const LAB = '/network/lab';
/** Rows that block Use and Change; the engine decides them, the page only shows them. */
export const blocking = (row: LabRow) => ['duplicate', 'other_card', 'invalid', 'missing'].includes(row.status);
