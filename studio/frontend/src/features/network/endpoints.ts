import type { Ied, NetworkSettings } from '../../api/types';

/** True when this IED really answers on the lab adapter. In lab mode an IED added after the
 * network setup has no lab address and still answers on this PC only (change 0102). */
export function onLabNetwork(ied: Pick<Ied, 'lab_ip'>, network: NetworkSettings) {
  return network.mode === 'lab' && !!ied.lab_ip;
}

/** The address a simulated IED listens on, mirroring the backend effective_endpoint. */
export function endpointOf(ied: Pick<Ied, 'ip' | 'port' | 'lab_ip'>, network: NetworkSettings) {
  return { host: onLabNetwork(ied, network) ? ied.lab_ip : ied.ip, port: ied.port };
}

/** Last part of an object reference, e.g. "gcbEvents" from "IED1LD0/LLN0$GO$gcbEvents". */
export function shortBlock(reference: string) {
  return reference.split('$').pop() || reference;
}
