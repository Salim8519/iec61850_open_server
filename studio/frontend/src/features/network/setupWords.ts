import type { Adapter, PlanPreview } from '../../api/network';

const KIND_WORD: Record<string, string> = { ethernet: 'Ethernet', wifi: 'Wi-Fi', virtual: 'Virtual',
  virtual_lab: 'Windows loopback adapter' };

/** Network cards worth offering: never VPN or Bluetooth. */
/** The picker's first choice: keep the IEDs on their private loopback addresses (this PC only). */
export const LOOPBACK_CHOICE = 'loopback';

export const usableAdapters = (adapters: Adapter[] | null) =>
  (adapters || []).filter(item => !['vpn', 'bluetooth'].includes(item.kind));

/** "Ethernet 2 · Ethernet · 192.168.230.5/24 · no cable" — everything needed to pick the right card. */
export function adapterLabel(adapter: Adapter) {
  const address = adapter.addresses.map(item => `${item.ip}/${item.prefix}`).join(', ') || 'no IPv4 address';
  return [adapter.name, KIND_WORD[adapter.kind] ?? adapter.kind, address,
    ...(adapter.status === 'Up' ? [] : ['no cable']), ...(adapter.default_route ? ['internet'] : [])].join(' · ');
}

/** One plain sentence for the whole change list; the exact list stays one click away. */
export function changeSentence(preview: PlanPreview) {
  const adds = preview.items.filter(item => item.action === 'add');
  const aliases = adds.filter(item => item.kind === 'alias').length;
  const rules = adds.length - aliases;
  if (!adds.length) return `Nothing changes in Windows: these addresses are already on “${preview.adapter.name}”.`;
  // The rule opens the IEDs' own MMS ports (a project may use other ports than 102), as the exact list says.
  const ports = adds.find(item => item.kind === 'firewall')?.ports ?? [102];
  const portWords = `${ports.length === 1 ? 'port' : 'ports'} ${ports.join(', ')}`;
  const parts = [aliases ? `add ${aliases} ${aliases === 1 ? 'address' : 'addresses'} to “${preview.adapter.name}”` : '',
    rules ? `allow ${portWords} for them in Windows Firewall` : ''].filter(Boolean).join(' and ');
  return `ScadaBench will ${parts}. Windows asks once. The addresses disappear when Windows restarts, `
    + 'or remove them earlier with Clean up.';
}

/** The one button of the page: "Put 4 IEDs on “Ethernet 2”", or "Put IEDs on this card" before a card is checked. */
export function putLabel(count: number, adapterName?: string) {
  const what = count ? `${count} ${count === 1 ? 'IED' : 'IEDs'}` : 'IEDs';
  return `Put ${what} on ${adapterName ? `“${adapterName}”` : 'this card'}`;
}
