import type { Node, Edge } from '@xyflow/react';

export type EquipmentKind = 'CB' | 'DS' | 'ES' | 'CT' | 'VT' | 'SOURCE' | 'RELAY' | 'TRIP_RELAY' | 'PUSH_BUTTON' | 'JUNCTION' | 'INFEED'
  | 'MEASUREMENT_LIST' | 'POINT_PANEL' | 'TAP_TRANSFORMER';
export type InfeedType = 'line' | 'transformer' | 'grid';
export type SignalBinding = { ied_id: string; report_ref: string | null; ref: string };
export type StateMapping =
  | { mode: 'direct'; source: SignalBinding }
  | { mode: 'single_to_dbpos'; source: SignalBinding }
  | { mode: 'contacts_to_dbpos'; open: SignalBinding; closed: SignalBinding };
export type ControlBinding = { ied_id: string; ref: string };
export type RelayStatus = { id: string; label: string; source: SignalBinding };
/** One measurement list row (ADR-0090): shown = raw × scale + offset, `decimals` places, then unit. */
export type Measurement = { id: string; label: string; source: SignalBinding;
  scale: number; offset: number; unit: string; decimals: number };
/** One point panel row (ADR-0102). status = Boolean signal, double = Dbpos signal, value = numeric MX signal
 *  (all three read `source`); command = SPC or DPC control (reads `control`). unit, scale, offset, decimals
 *  and pinned belong to value rows; `healthy` is the raw value "All healthy" writes. */
export type PointType = 'status' | 'double' | 'command' | 'value';
export type PanelPoint = { id: string; label: string; group: string; type: PointType;
  source: SignalBinding | null; control: ControlBinding | null;
  unit: string; scale: number; offset: number; decimals: number; pinned: boolean; healthy: boolean | number | null };
/** Simulated voltage regulator (90) of a tap-changer transformer (ADR-0109); runs only on simulated IEDs. */
export type AvrSettings = { enabled: boolean; target_kv: number; bandwidth_pct: number; delay_s: number;
  step_pct: number; hv_rated_kv: number; lv_rated_kv: number; source_kv: number };
/** On-load tap changer of a tap-changer transformer (ADR-0109): tap position, raise/lower command (one BSC/ISC
 *  "step" control or an SPC "spc_pair"), optional Auto/Manual status and command, optional LV voltage. */
export type TapCommandMode = 'none' | 'step' | 'spc_pair';
export type TapChanger = { tap_source: SignalBinding | null; command_mode: TapCommandMode;
  step_control: ControlBinding | null; raise_control: ControlBinding | null; lower_control: ControlBinding | null;
  mode_source: SignalBinding | null; mode_control: ControlBinding | null; auto_value: number; manual_value: number;
  voltage_source: SignalBinding | null; min_tap: number; max_tap: number; nominal_tap: number; avr: AvrSettings };
export type EquipmentData = {
  label: string; kind: EquipmentKind; rotation: number;
  state_mapping?: StateMapping | null; control_binding?: ControlBinding | null;
  relay_statuses?: RelayStatus[];
  /** Measurement list only; other kinds never carry it. */
  measurements?: Measurement[];
  /** Point panel only: the one IED and its points (up to 300); other kinds never carry them. */
  ied_id?: string | null; points?: PanelPoint[];
  /** Tap-changer transformer only (ADR-0109). */
  tap_changer?: TapChanger;
  trip_command_polarity?: 'operate_false' | 'operate_true';
  trip_state_polarity?: 'operated_false' | 'operated_true';
  push_mode?: 'spc_momentary' | 'latching_status' | 'self_resetting'; push_value?: boolean;
  push_rest_value?: boolean; push_reset_ms?: number;
  initial_position?: 'open' | 'closed';
  /** CB, DS and ES only, after options were sent once (ADR-0108). */
  command_options?: CommandOptions;
  active_color?: string | null; inactive_color?: string | null;
  /** Infeed only (drawing; never sent to an IED or SCL). */
  infeed_type?: InfeedType; infeed_state?: 'live' | 'dead';
};
export type EquipmentNode = Node<EquipmentData, 'equipment'>;
export type Diagram = { nodes: EquipmentNode[]; edges: Edge[] };
export type Signal = {
  ref: string; fc: string; kind: string; value: number | string | boolean;
  writable: boolean; quality: number; timestamp: number; description: string;
  quality_ref?: string | null; timestamp_ref?: string | null;
  enums: Record<string, number>;
};
/** model is the IEC 61850 ctlModel 0..4 (status-only, direct-normal, sbo-normal, direct-enhanced, sbo-enhanced). */
export type Control = { ref: string; status: string; cdc: 'SPC' | 'DPC'; ctl_val_kind: 'BOOLEAN';
  model: number; sbo_timeout_ms: number; delay_ms: number; blocked: boolean };
export type ControlModel = 'status-only' | 'direct-normal' | 'sbo-normal' | 'direct-enhanced' | 'sbo-enhanced';
/** One typed control of an IED (ADR-0104 `extended_controls`, read by the Go engine from the stored SCL and
 *  served by GET /controls/typed, never part of the IED payload). model is the ctlModel 0..4 as in `controls`;
 *  min, max and step bound INC and APC values; labels map each ENC choice to its number, in the model's order. */
export type ExtendedControl = { ref: string; status: string; cdc: 'INC' | 'ENC' | 'APC' | 'BAC' | 'BSC' | 'ISC';
  ctl_val_kind: string; ctl_val_ref: string; value_kind: 'int' | 'enum' | 'float' | 'step'; model: number;
  sbo_timeout_ms: number; min: number | null; max: number | null; step: number | null; labels: Record<string, number>;
  trans_ind: string | null };
/** GET /controls/typed?ied_id= (ADR-0104): the SPC/DPC list and the typed list of one IED. */
export type TypedControlList = { controls: Control[]; extended_controls: ExtendedControl[] };
/** Options of one command (ADR-0108); a breaker, disconnector or earth switch remembers the last ones sent. */
export type CommandOptions = { test: boolean; interlock_check: boolean; synchro_check: boolean; orcat: number; orident: string };
export type Ied = {
  id: string; name: string; ip: string; port: number; state: string; signal_count: number;
  controls: Control[]; extended_controls?: ExtendedControl[]; warnings: string[];
  validation?: { status: 'valid' | 'invalid' | 'unsupported'; schema: string | null;
    errors: { line: number; message: string }[] };
  reports: { name: string; ref: string; dataset: string; dataset_ref: string | null; buffered: boolean;
    signal_count: number; unresolved: { ref: string; fc: string }[] }[];
  role: IedRole; live: LiveSettings; lab_ip: string; goose_publish: boolean;
  scl_address: { ip: string; subnet: string; gateway: string; access_point: string } | null;
  goose: { publishes: GooseControl[]; subscribes: (GooseControl & { publisher: string;
    inputs: { int_addr: string; desc: string; source: string }[] })[]; diagnostics: string[] };
  goose_state?: { active?: boolean; requested?: boolean; reason?: string };
};
export type IedRole = 'simulated' | 'real' | 'off';
export type LiveSettings = { ip: string; port: number; access: 'auto_reports' | 'polling'; poll_ms: number };
export type GooseControl = { name: string; gocb_ref: string; go_id: string; dataset_ref: string; app_id: number;
  mac: string; vlan_id: number; vlan_priority: number; conf_rev: number };
export type NetworkSettings = { mode: 'loopback' | 'lab'; adapter_guid: string; adapter_label: string;
  goose_reach?: 'inside' | 'outside' };
export type Project = {
  id: string; name: string; session_id: string; revision: number;
  ieds: Ied[]; diagram: Diagram; runtime: string; network: NetworkSettings;
  /** IEDs in the current run; one IED can run alone (ADR-0067). */
  running_ieds?: string[];
  scenarios: Scenario[];
  test_panel: TestControl[];
  sources: { ied_id: string; ref: string; spec: SourceSpec }[];
  startup_recovery?: { status: 'restored' | 'unreadable'; message: string;
    details?: string; backup?: string } | null;
  migration_notice?: { message: string; details: string[]; backup: string } | null;
};
export type SourceSpec = { mode: string; offset?: number; amplitude?: number; period?: number;
  seed?: number; overridden?: boolean; interpolation?: string; repeat?: boolean };
export type TestControl = { id: string; label: string; ied_id: string; ref: string;
  report_ref: string | null; duration_ms: number; active_value: boolean };
export type ScenarioStep = { id: string; at: number; ied_id: string; ref: string;
  action: 'set' | 'assert'; value: Signal['value'] };
export type Scenario = { id: string; name: string; enabled: boolean; steps: ScenarioStep[] };
export type StudioEvent = {
  id: number; kind: string; timestamp: number; ied_id?: string; ref?: string;
  status_ref?: string; node_id?: string; value?: Signal['value'] | null; state?: string; message?: string;
  [key: string]: unknown;
};
