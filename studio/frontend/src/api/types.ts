import type { Node, Edge } from '@xyflow/react';

export type EquipmentKind = 'CB' | 'DS' | 'ES' | 'CT' | 'VT' | 'TRANSFORMER' | 'SOURCE' | 'LOAD' | 'RELAY' | 'TRIP_RELAY' | 'PUSH_BUTTON' | 'BUSBAR' | 'JUNCTION';
export type SignalBinding = { ied_id: string; report_ref: string | null; ref: string };
export type StateMapping =
  | { mode: 'direct'; source: SignalBinding }
  | { mode: 'single_to_dbpos'; source: SignalBinding }
  | { mode: 'contacts_to_dbpos'; open: SignalBinding; closed: SignalBinding };
export type ControlCompatibility = 'standard' | 'concen_inverted_sbow';
export type ControlBinding = { ied_id: string; ref: string; compatibility?: ControlCompatibility };
export type RelayStatus = { id: string; label: string; source: SignalBinding };
export type EquipmentData = {
  label: string; kind: EquipmentKind; rotation: number; width?: number | null;
  state_mapping?: StateMapping | null; control_binding?: ControlBinding | null;
  relay_statuses?: RelayStatus[];
  trip_command_polarity?: 'operate_false' | 'operate_true';
  trip_state_polarity?: 'operated_false' | 'operated_true';
  push_mode?: 'spc_momentary' | 'latching_status' | 'self_resetting'; push_value?: boolean;
  push_rest_value?: boolean; push_reset_ms?: number;
  initial_position?: 'open' | 'closed';
  active_color?: string | null; inactive_color?: string | null;
};
export type EquipmentNode = Node<EquipmentData, 'equipment'>;
export type Diagram = { nodes: EquipmentNode[]; edges: Edge[] };
export type Signal = {
  ref: string; fc: string; kind: string; value: number | string | boolean;
  writable: boolean; quality: number; timestamp: number; description: string;
  quality_ref?: string | null; timestamp_ref?: string | null;
  enums: Record<string, number>;
};
export type Control = { ref: string; status: string; cdc: 'SPC' | 'DPC'; ctl_val_kind: 'BOOLEAN';
  model: number; sbo_timeout_ms: number; delay_ms: number; blocked: boolean };
export type Ied = {
  id: string; name: string; ip: string; port: number; state: string; signal_count: number;
  controls: Control[]; warnings: string[];
  validation?: { status: 'valid' | 'invalid' | 'unsupported'; schema: string | null;
    errors: { line: number; message: string }[] };
  reports: { name: string; ref: string; dataset: string; dataset_ref: string | null; buffered: boolean;
    signal_count: number; unresolved: { ref: string; fc: string }[] }[];
};
export type Project = {
  id: string; name: string; session_id: string; revision: number;
  ieds: Ied[]; diagram: Diagram; runtime: string;
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
