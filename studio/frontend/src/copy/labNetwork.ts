import type { LabPing, LabRealCard, LabStatus } from '../api/lab';
import type { Status } from './status';

// Words of the "This PC lab network" card (ADR-0120). The hint under each pill comes from the engine.
export const LAB_ROW_STATUS: Record<string, Status> = {
  ok: { label: 'Ready', tone: 'good', hint: '' },
  same_range: { label: 'Same range as a real card', tone: 'info', hint: '' },
  real_device: { label: 'A real device uses this address', tone: 'warning', hint: '' },
  checking: { label: 'Checking…', tone: 'waiting', hint: '' },
  duplicate: { label: 'Used twice', tone: 'problem', hint: '' },
  other_card: { label: 'Address of this PC', tone: 'problem', hint: '' },
  invalid: { label: 'Address not valid', tone: 'problem', hint: '' },
  missing: { label: 'No address', tone: 'problem', hint: '' },
};

export const LAB_COPY = {
  intro: 'Test your IEDs at their real addresses from the SCL file, without a VM. Ping, MMS and GOOSE work; '
    + 'nothing goes out on your Ethernet or Wi-Fi.',
  create: 'Create the lab network card',
  createHint: 'Windows asks for permission once. ScadaBench adds the "ScadaBench Lab" card (the Windows loopback adapter, '
    + 'no download, no restart) and a small service, so addresses can change later without asking again.',
  creating: 'Waiting for Windows: approve the permission prompt (UAC). It may be behind this window.',
  repair: 'Repair',
  editHint: 'Type another address and press Enter; it changes at once, even while running. The SCL file is never changed.',
  use: 'Use the lab network',
  useHint: 'No Windows prompt: the lab network service adds the addresses.',
  stopFirst: 'Stop the run to switch the network.',
  show: (count: number) => `Show the ${count} ${count === 1 ? 'address' : 'addresses'}`,
  remove: 'Remove the lab network card',
  removeHint: 'Removes the card, its addresses and the service (one Windows prompt). Your IEDs go back to this PC only.',
  ping: 'Ping',
  backToFile: 'Use file address',
  onceHint: 'First time only: Windows asks once to add the card.',
  // The network card list (ADR-0121): the two choices that keep everything on this PC.
  pickLoopback: 'This PC only (loopback): IEDs talk to each other here',
  tagLoopback: 'Simplest',
  pickLab: 'ScadaBench Lab: each IED at its own SCL address',
  tagLab: 'No VM · ping works',
  pickPlaceholder: 'Choose where your IEDs answer…',
};

/** The card's one line: what the lab network does for this project right now. */
export function labSummary(status: LabStatus) {
  const count = status.rows.length;
  const ieds = `${count} ${count === 1 ? 'IED' : 'IEDs'}`;
  const attention = status.rows.filter(row => !['ok', 'same_range', 'checking'].includes(row.status)).length;
  const note = attention ? ` ${attention} ${attention === 1 ? 'needs' : 'need'} attention.` : '';
  if (status.active) return `In use: ${ieds} answer at their own addresses on this PC.${note}`;
  return `Ready: ${ieds} can answer at their SCL addresses on this PC, without a VM.${note}`;
}

/** "Ethernet 192.168.40.111/24 · no cable" for the line about this PC's real cards. */
export function realCardLine(card: LabRealCard) {
  const networks = card.networks.length ? card.networks.join(', ') : 'no address';
  return `${card.name} ${networks}${card.connected ? '' : ' · no cable'}`;
}

/** One ping answer in the words of Windows ping. */
export function pingWords(address: string, ping: LabPing | 'failed') {
  if (ping === 'failed' || !ping.ok) return `No reply from ${address}`;
  return `Reply from ${address}: time${ping.ms < 1 ? '<1' : '=' + ping.ms}ms TTL=${ping.ttl}`;
}
