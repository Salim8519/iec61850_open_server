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
  if (!adds.length) return `Nothing changes in Windows: these addresses are already on “${preview.adapter.name}”.`;
  // The rules open the IEDs' own MMS ports (a project may use other ports than 102) and the SNMP
  // devices' UDP ports, as the exact list says.
  const mms = adds.find(item => item.kind === 'firewall');
  const snmp = adds.find(item => item.kind === 'snmp_firewall');
  const portWords = (ports: number[]) => `${ports.length === 1 ? 'port' : 'ports'} ${ports.join(', ')}`;
  const rules = [mms ? `${portWords(mms.ports ?? [102])}${snmp ? ' for the IEDs' : ''}` : '',
    snmp ? `UDP ${portWords(snmp.ports ?? [161])} for the SNMP devices` : ''].filter(Boolean).join(' and ');
  const parts = [aliases ? `add ${aliases} ${aliases === 1 ? 'address' : 'addresses'} to “${preview.adapter.name}”` : '',
    rules ? `allow ${rules}${mms && !snmp ? ' for them' : ''} in Windows Firewall` : ''].filter(Boolean).join(' and ');
  return `ScadaBench will ${parts}. Windows asks once. The addresses disappear when Windows restarts, `
    + 'or remove them earlier with Clean up.';
}

/** The one button of the page: "Put 4 IEDs and 2 SNMP devices on “Ethernet 2”", or "Put IEDs on this card"
 *  before a card is checked. */
export function putLabel(count: number, adapterName?: string, snmp = 0) {
  const ieds = count ? `${count} ${count === 1 ? 'IED' : 'IEDs'}` : '';
  const devices = snmp ? `${snmp} SNMP ${snmp === 1 ? 'device' : 'devices'}` : '';
  const what = [ieds, devices].filter(Boolean).join(' and ') || 'IEDs';
  return `Put ${what} on ${adapterName ? `“${adapterName}”` : 'this card'}`;
}

/** The page title names SNMP devices once the project has any (they follow the IEDs, ADR-0092). */
export function setupTitle(preview: PlanPreview | null) {
  return preview?.rows.some(row => row.kind === 'snmp')
    ? 'Put your published IEDs and SNMP devices on a network' : 'Put your published IEDs on a network';
}

/** One line under the network mode: SNMP devices follow the IEDs (ADR-0092). */
export function snmpFollowLine(count: number, lab: boolean) {
  const one = count === 1;
  const devices = `Your ${count} SNMP ${one ? 'device follows' : 'devices follow'} the IEDs`;
  return lab ? `${devices}: ${one ? 'it answers' : 'each answers'} at its own address on this card.`
    : `${devices} and ${one ? 'stays' : 'stay'} on this PC too.`;
}
