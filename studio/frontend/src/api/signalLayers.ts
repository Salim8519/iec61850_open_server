import { api } from './client';

/** GOOSE link map and MMS client list from the backend (spec signal-layers). The browser only displays them. */
export type LinkStatus = 'declared' | 'remote' | 'disabled_in_file' | 'no_dataset' | 'no_address' | 'not_goose' | 'not_sendable';
export type LinkLive = { state: string; problem?: string; st_num?: number; observed_at?: string } | null;
export type GooseInput = { int_addr: string; desc: string; source: string };
export type GooseLink = {
  id: string; gocb_ref: string | null; publisher: string; subscriber: string; block: string;
  status: LinkStatus; remote: boolean; match: string; mac?: string; inputs: GooseInput[]; live: LinkLive;
};
export type GooseIed = {
  name: string; ied_id: string | null; role: string | null; in_project: boolean; remote: boolean;
  desc: string; manufacturer: string;
};
export type GooseBlock = {
  publisher: string; ld_inst: string; name: string; gocb_ref: string; go_id: string; dataset: string;
  dataset_ref: string; conf_rev: number | null; app_id: number | null; mac: string; status: string;
  label: string; members: string[]; member_refs: string[];
};
export type GooseDiagnostic = {
  level: 'problem' | 'warning' | 'info' | 'hint'; code: string; ied: string | null; gocb_ref: string | null;
  message: string; hint: string;
};
export type GooseSummary = {
  sends_to: number; gets_from: string[]; not_sent: number; remote: number; not_heard: number; problems: number;
};
export type GooseLinkMap = {
  ieds: GooseIed[]; blocks: GooseBlock[]; links: GooseLink[]; diagnostics: GooseDiagnostic[];
  summary: Record<string, GooseSummary>; running: boolean;
};
export type MmsClient = {
  peer: string; ip: string; own: boolean; this_pc: boolean; reading: boolean;
  reads: { node: string; count: number }[]; more_nodes: number; total_reads: number; writes: number;
  rcbs: string[]; connected_s: number;
};

export const getMmsAccess = () => api<{ running: boolean; ieds: Record<string, MmsClient[]> }>('/mms/access');
