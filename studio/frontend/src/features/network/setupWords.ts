import type { Adapter, PlanPreview } from '../../api/network';

const KIND_WORD: Record<string, string> = { ethernet: 'Ethernet', wifi: 'Wi-Fi', virtual: 'Virtual',
  virtual_lab: 'Windows loopback adapter' };

/** Network cards worth offering: never VPN or Bluetooth. */
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
  const parts = [aliases ? `add ${aliases} ${aliases === 1 ? 'address' : 'addresses'} to “${preview.adapter.name}”` : '',
    rules ? 'allow port 102 for them in Windows Firewall' : ''].filter(Boolean).join(' and ');
  return `ScadaBench will ${parts}. Windows asks once. The addresses disappear when Windows restarts, `
    + 'or remove them earlier with Clean up.';
}
