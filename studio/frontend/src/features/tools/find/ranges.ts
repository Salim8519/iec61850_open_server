import type { Adapter } from '../../../api/network';

export type RangeChoice = { key: string; label: string; range: string; own: string };

const toNumber = (ip: string) => ip.split('.').reduce((total, part) => total * 256 + Number(part), 0);
const toText = (value: number) => [24, 16, 8, 0].map(shift => Math.floor(value / 2 ** shift) % 256).join('.');

/** The adapter's own network, narrowed to a /24 at most (the backend enforces the same limit). */
export function networkOf(ip: string, prefix: number): string {
  const bits = Math.max(24, Math.min(32, prefix));
  const size = 2 ** (32 - bits);
  return `${toText(Math.floor(toNumber(ip) / size) * size)}/${bits}`;
}

/** One choice per IPv4 address of every connected adapter, plus the simulated IEDs on this PC. */
export function rangeChoices(adapters: Adapter[], thisPc: string): RangeChoice[] {
  const choices = adapters.filter(adapter => adapter.status === 'Up').flatMap(adapter => adapter.addresses
    .filter(address => address.ip && !address.ip.startsWith('169.254.') && !address.ip.startsWith('127.'))
    .map(address => ({ key: `${adapter.guid}|${address.ip}`, own: address.ip,
      label: `${adapter.name} · ${address.ip}/${address.prefix}`, range: networkOf(address.ip, address.prefix) })));
  return [...choices, { key: 'this-pc', label: thisPc, range: '127.50.0.0/24', own: '' }];
}
