// Response shapes of the network, live and GOOSE endpoints.
export type AdapterAddress = { ip: string; prefix: number; skip_as_source: boolean; origin: string };
export type Adapter = { guid: string; name: string; description: string; status: string; index: number;
  speed: string; kind: string; default_route: boolean; addresses: AdapterAddress[] };
export type PlanRow = { ied_id: string; name: string; scd_ip: string; proposed: string; prefix: number | null;
  status: string; hint: string; alternative?: { ip: string; prefix: number };
  /** An SNMP device row (ADR-0092): its saved address may be shown instead of a new free one. */
  kind?: 'snmp'; saved?: boolean };
export type PlanItem = { kind: 'alias' | 'firewall' | 'snmp_firewall' | 'gateway_firewall' | 'unblock'; action: 'add' | 'keep' | 'remove'; owned: boolean;
  description: string; ip?: string; name?: string;
  /** Firewall rule: the IEDs' TCP ports (normally 102), or the SNMP devices' UDP ports (normally 161). */
  ports?: number[] };
export type PlanPreview = { adapter: Adapter; rows: PlanRow[]; items: PlanItem[]; fingerprint: string;
  ready: boolean; needs_admin: boolean };
export type ApplyResult = { applied?: boolean; message: string;
  results: { ok: boolean; message: string; description: string }[] };
export type NpcapStatus = { installed: boolean; download: string; message: string };
export type GooseWatch = { key: string; gocb_ref: string; app_id: number; mac: string; go_id: string;
  dataset_ref: string; conf_rev: number; state: string; problem: string; frames: number;
  st_num?: number | null; sq_num?: number | null; age_ms?: number; origin?: 'app' | 'wire'; src_mac?: string;
  publisher?: { ied_id: string | null; name: string; role: string };
  subscribers: { ied_id: string; name: string; lgos: string | null; inputs: number }[] };
export type GooseConflict = { ied_id: string; gocb_ref: string; src_mac: string; sharing: boolean;
  choice: 'stop' | 'continue' | null };
export type GooseEngineState = { running: boolean; uplink?: string; share?: string; message?: string;
  conflicts: GooseConflict[] };
export type GooseUplink = { guid: string; label: string; kind: string; connected: boolean; auto: boolean;
  why: string; reason: string; warning?: string };
/** GET /goose/share: everything the "Where GOOSE goes" card and its warning need (ADR-0073, ADR-0122).
 * goes_out: the user shared GOOSE and an adapter is there; alone: nothing came back while a GOOSE is missing. */
export type GooseShare = { reach: 'inside' | 'outside'; uplink: GooseUplink; uplink_guid: string;
  choices: GooseUplink[]; risk: { level: 'high' | 'warning'; reasons: string[] }; npcap: boolean;
  running: boolean; state: string; message: string; conflicts: GooseConflict[]; goes_out: boolean; alone: boolean };
export type GooseOverview = { running: boolean; state: string; message: string; npcap: NpcapStatus;
  engine?: GooseEngineState; watches: GooseWatch[]; unknown: { gocb_ref: string; app_id: number; mac: string; dataset: string }[] };
export type AccessPlan = { reports: string[]; polling_datasets: string[]; notes: string[] };
export type LiveSession = { ied_id: string; name: string; state: string; host: string; port: number;
  endpoint_source: 'scd' | 'override' | 'missing'; message: string; access: AccessPlan | null };
export type CompareRow = { ref: string; fc: string; category: string; scd_type: string | null;
  device_type: string | null; hint: string };
export type Comparison = { summary: Record<string, number>; signals: CompareRow[];
  datasets: { ref: string; status: string; missing: string[]; extra: string[] }[];
  reports: { ref: string; status: string }[] };
/** GET /network/plan/progress (Go engine, change 0150): the running address check, step by step. */
export type CheckProgress = { run: number; running: boolean; asked: number; answered: number;
  steps: { key: string; label: string; state: 'waiting' | 'running' | 'done'; ms: number }[];
  rows: { id: string; name: string; address: string; state: 'checking' | 'free' | 'in_use' | 'not_checked' }[] };
