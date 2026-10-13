import type { Ied, NetworkSettings } from '../../api/types';

/** The address a simulated IED listens on, mirroring the backend effective_endpoint. */
export function endpointOf(ied: Pick<Ied, 'ip' | 'port' | 'lab_ip'>, network: NetworkSettings) {
  const host = network.mode === 'lab' && ied.lab_ip ? ied.lab_ip : ied.ip;
  return { host, port: ied.port };
}

/** Last part of an object reference, e.g. "gcbEvents" from "IED1LD0/LLN0$GO$gcbEvents". */
export function shortBlock(reference: string) {
  return reference.split('$').pop() || reference;
}
