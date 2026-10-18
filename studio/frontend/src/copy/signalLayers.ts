import type { GooseLink, MmsClient } from '../api/signalLayers';
import { gooseStatus, type Status } from './status.ts';

/** Words for the SLD signal layers and the Devices GOOSE tables (spec signal-layers). */
export const LAYER_COPY = {
  goose: { label: 'GOOSE messages', hint: 'Show which IEDs send GOOSE to which, with one quiet dot when a message carries a new state.' },
  readers: { label: 'Who is reading', hint: 'Show which SCADA or test programs read each simulated IED over MMS, by their IP address.' },
};

const NOT_SENT: Record<string, Status> = {
  disabled_in_file: { label: 'Not sent (disabled in file)', tone: 'off', hint: 'The file switches this GOOSE block off, so the device never sends it. Normal for spare inputs.' },
  no_dataset: { label: 'Not sent (no dataset)', tone: 'off', hint: 'The block has no dataset in the file, so it carries nothing.' },
  no_address: { label: 'Not sent (no address)', tone: 'warning', hint: 'The file gives this block no GOOSE network address (GSE).' },
  not_goose: { label: 'Not GOOSE', tone: 'off', hint: 'This control block is not a GOOSE block.' },
  not_sendable: { label: 'Cannot be sent', tone: 'warning', hint: 'The block is written in a way the engine cannot send. See File check details.' },
  remote: { label: 'Outside this project', tone: 'info', hint: 'The sending IED is in none of this project\'s files. Import its file or add it as a real device to supervise it.' },
};

/** One status for a GOOSE link: why nothing is sent, or what the listener last saw. */
export function linkStatus(link: GooseLink, running: boolean): Status {
  if (NOT_SENT[link.status]) return NOT_SENT[link.status];
  if (!running) return { label: 'Declared in SCD', tone: 'off', hint: 'Press Run to see whether it really arrives.' };
  if (!link.live) return { label: 'Not watched', tone: 'off', hint: 'GOOSE is watched only in lab mode with Npcap installed.' };
  const status = gooseStatus(link.live.state, link.live.problem);
  return link.live.state === 'waiting' ? { ...status, label: 'Not heard yet' } : status;
}

export function directionWords(link: GooseLink, name: string): string {
  return link.publisher === name ? `Sends to ${link.subscriber}` : `Gets from ${link.publisher}`;
}

/** Short Devices list hint, for example "↑ sends to 2 · ↓ gets from C09 · ⚠ 1 not heard". */
export function gooseHint(summary: { sends_to: number; gets_from: string[]; not_heard: number; problems: number } | undefined,
  names: Record<string, string> = {}): string {
  if (!summary) return '';
  const parts: string[] = [];
  if (summary.sends_to) parts.push(`↑ sends to ${summary.sends_to}`);
  if (summary.gets_from.length) {
    const first = names[summary.gets_from[0]] || summary.gets_from[0];
    parts.push(`↓ gets from ${summary.gets_from.length === 1 ? first : `${summary.gets_from.length}`}`);
  }
  if (summary.problems) parts.push(`⚠ ${summary.problems} problem${summary.problems > 1 ? 's' : ''}`);
  if (summary.not_heard) parts.push(`⚠ ${summary.not_heard} not heard`);
  return parts.join(' · ');
}

/** Readers of one IED, without ScadaBench's own command client. */
export const externalClients = (clients: MmsClient[] = []) => clients.filter(client => !client.own);

export function readerBadge(clients: MmsClient[] = []): string {
  const count = externalClients(clients).length;
  return count ? `◉ ${count} read` : '';
}

export function clientLine(client: MmsClient): string {
  const where = client.this_pc ? ' (this PC)' : '';
  const reads = client.reads.slice(0, 3).map(item => item.node.split('/').pop()).join(', ');
  const more = client.reads.length > 3 || client.more_nodes ? ' …' : '';
  const reports = client.rcbs.length ? `; reports on ${client.rcbs.join(', ')}` : '';
  return `${client.ip}${where}: ${reads ? `reads ${reads}${more}` : 'connected'}${reports}`;
}
