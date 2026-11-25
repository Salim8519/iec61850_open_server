import type { Adapter } from '../../../api/network';

/** One network the chosen interface is on: its own address and the range Find IEDs would check. */
export type RangeNetwork = { own: string; range: string; label: string };
/** One interface (or "This PC"). `range` is its first network; `networks` lists all of them. */
export type RangeChoice = { key: string; label: string; range: string; own: string; networks: RangeNetwork[] };

const toNumber = (ip: string) => ip.split('.').reduce((total, part) => total * 256 + Number(part), 0);
const toText = (value: number) => [24, 16, 8, 0].map(shift => Math.floor(value / 2 ** shift) % 256).join('.');

/** The adapter's own network, narrowed to a /24 at most (the backend enforces the same limit). */
export function networkOf(ip: string, prefix: number): string {
  const bits = Math.max(24, Math.min(32, prefix));
  const size = 2 ** (32 - bits);
  return `${toText(Math.floor(toNumber(ip) / size) * size)}/${bits}`;
}

const usable = (ip: string) => Boolean(ip) && !ip.startsWith('169.254.') && !ip.startsWith('127.');

/** The distinct networks of one adapter, in the adapter's own address order. */
function networksOf(adapter: Adapter): RangeNetwork[] {
  const seen = new Set<string>();
  return adapter.addresses.filter(address => usable(address.ip)).flatMap(address => {
    const range = networkOf(address.ip, address.prefix);
    if (seen.has(range)) return [];
    seen.add(range);
    return [{ own: address.ip, range, label: `${address.ip}/${address.prefix}` }];
  });
}

/** One choice per connected interface, however many addresses it has, plus the simulated IEDs
 *  on this PC (change 0237). An interface's extra networks are offered beside the range box. */
export function rangeChoices(adapters: Adapter[], thisPc: string): RangeChoice[] {
  const choices = adapters.filter(adapter => adapter.status === 'Up').flatMap(adapter => {
    const networks = networksOf(adapter);
    if (!networks.length) return [];
    const extra = networks.length > 1 ? ` (${networks.length} networks)` : '';
    return [{ key: adapter.guid, label: `${adapter.name}${extra}`, range: networks[0].range, own: networks[0].own, networks }];
  });
  const local = { own: '', range: '127.50.0.0/24', label: '127.50.0.0/24' };
  return [...choices, { key: 'this-pc', label: thisPc, range: local.range, own: '', networks: [local] }];
}

/** The remembered choice, also when it was saved per address before 0237 ("guid|ip"). */
export function rememberedChoice(choices: RangeChoice[], saved: string | undefined): RangeChoice | undefined {
  if (!saved) return undefined;
  return choices.find(item => item.key === saved) ?? choices.find(item => item.key === saved.split('|')[0]);
}
