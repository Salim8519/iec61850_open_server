// Go-engine routes of the virtual machine host key (ADR-0099), under /api/v1/license so the license
// gate never blocks them. The frozen Python engine answers 404, and the screens then stay hidden.
import { api } from './client';
import type { ApplyResult } from './network';

export type VmHostStatus = {
  /** This PC's license can be shared; false comes with a plain-words `reason`. */
  available: boolean;
  reason: string | null;
  /** The masked host key, only when available. */
  key: string | null;
  /** The "Let my virtual machines use this license" switch (the Run value exists). */
  enabled: boolean;
  /** The helper program is running in this Windows session. */
  running: boolean;
  helper_found: boolean;
  port: number;
};
export type VmFirewallItem = { kind: string; action: 'add' | 'keep'; description: string };
export type VmFirewallPreview = { items: VmFirewallItem[]; fingerprint: string; open: boolean; needs_admin: boolean };

export const getVmHost = () => api<VmHostStatus>('/license/host-key');
export const setVmHost = (enabled: boolean) => api<VmHostStatus>('/license/host-key', 'PUT', { enabled });
export const getVmFirewall = () => api<VmFirewallPreview>('/license/host-key/firewall');
export const applyVmFirewall = (fingerprint: string) =>
  api<ApplyResult>('/license/host-key/firewall', 'POST', { fingerprint, confirm: true });
/** "<64-hex Machine ID>.<host key>" for the offline activation page. */
export const getActivationCode = () => api<{ code: string }>('/license/activation-code');
