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
  remote: { label: 'Outside project', tone: 'info', hint: 'The sender is not in this project, so nothing sends this GOOSE here. Import its CID, or connect the real device on the lab network.' },
};

/** Plain words when nothing can be heard (change 0097; replaces the unclear old wording). */
export const NOT_HEARD = {
  needsLab: { label: 'Needs lab network', tone: 'off', hint: 'Not received because GOOSE only travels on a lab network (a real network cable). In "This PC only" mode nothing can arrive. Set up the lab network to see it.' },
  noListener: { label: 'Not checked', tone: 'off', hint: 'ScadaBench checks arriving GOOSE only when the free Npcap driver is installed. Install Npcap, then press Run again.' },
} satisfies Record<string, Status>;

/** One status for a GOOSE link: why nothing is sent, or what the listener last saw. */
export function linkStatus(link: GooseLink, running: boolean, lab = false, outside = false): Status {
  if (NOT_SENT[link.status]) return NOT_SENT[link.status];
  if (outside) return NOT_SENT.remote;
  if (!running) return { label: 'Declared in SCD', tone: 'off', hint: 'Press Run to see whether it really arrives.' };
  if (!link.live) return lab ? NOT_HEARD.noListener : NOT_HEARD.needsLab;
  const status = gooseStatus(link.live.state, link.live.problem);
  return link.live.state === 'waiting' ? { ...status, label: 'Not heard yet' } : status;
}

export function directionWords(link: GooseLink, name: string): string {
  return link.publisher === name ? `Sends to ${link.subscriber}` : `Gets from ${link.publisher}`;
}

/** One compact sheet line: "← C11_Y120 · GoCB01" (gets) or "→ C12_Y130 · GoCB02" (sends), full names. */
export function linkLine(link: GooseLink, name: string): string {
  const arrow = link.publisher === name ? `→ ${link.subscriber}` : `← ${link.publisher}`;
  return `${arrow}${link.block ? ` · ${link.block}` : ''}`;
}

/** True when a running link can only be heard after the lab network is set up. */
export const needsLab = (link: GooseLink, running: boolean, lab: boolean) =>
  running && !lab && !link.live && !NOT_SENT[link.status];

/** Names of IEDs in the GOOSE map that are not in this project (senders shown only in the toolbar). */
export const outsideNames = (ieds: { name: string; in_project: boolean }[]) =>
  new Set(ieds.filter(ied => !ied.in_project).map(ied => ied.name));

export const OUTSIDE_COPY = {
  button: (count: number) => `${count} sender${count === 1 ? '' : 's'} outside project`,
  intro: 'These IEDs send GOOSE to IEDs in this project, but their files are not imported. Their messages appear as one dot from the canvas edge.',
  importCid: 'Import its CID',
  importHint: 'Import the CID (or the station SCD) of this IED so ScadaBench can simulate it, or connect the real device on the lab network.',
};

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
