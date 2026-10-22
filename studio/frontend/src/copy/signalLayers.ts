import type { GooseLink, MmsClient } from '../api/signalLayers';
import type { ChangeGroup } from '../equipment/signalChanges.ts';
import { symbolName } from '../equipment/signalOrigin.ts';
import { valueWords } from '../equipment/signalPulses.ts';
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
  remote: { label: 'Outside project', tone: 'info', hint: 'The sender is not in this project, so nothing sends this GOOSE here. Import its CID, or connect the real device to the network ScadaBench listens on.' },
};

/** Plain words while the GOOSE check has not reported yet. GOOSE runs inside ScadaBench in
 * every network mode (ADR-0073), so a lab network is no longer needed to hear it. */
export const NOT_HEARD = {
  notYet: { label: 'Not checked yet', tone: 'waiting', hint: 'ScadaBench starts checking GOOSE a moment after Run. If this stays, open Tools > Traffic.' },
} satisfies Record<string, Status>;

/** One status for a GOOSE link: why nothing is sent, or what the listener last saw. A sender
 * outside the project whose GOOSE does arrive shows that live state, not "Outside project" (change 0102). */
export function linkStatus(link: GooseLink, running: boolean, outside = false): Status {
  const remote = outside || link.status === 'remote';
  if (remote && running && link.live && link.live.state !== 'waiting' && link.live.state !== 'stopped') {
    const status = gooseStatus(link.live.state, link.live.problem);
    return { ...status, hint: `${status.hint} ${link.publisher} is not in this project; its GOOSE comes from the network.` };
  }
  if (NOT_SENT[link.status]) return NOT_SENT[link.status];
  if (outside) return NOT_SENT.remote;
  if (!running) return { label: 'Declared in SCD', tone: 'off', hint: 'Press Run to see whether it really arrives.' };
  if (!link.live) return NOT_HEARD.notYet;
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

/** Names of IEDs in the GOOSE map that are not in this project (senders shown only in the toolbar). */
export const outsideNames = (ieds: { name: string; in_project: boolean }[]) =>
  new Set(ieds.filter(ied => !ied.in_project).map(ied => ied.name));

export const OUTSIDE_COPY = {
  button: (count: number) => `${count} sender${count === 1 ? '' : 's'} outside project`,
  intro: 'These IEDs send GOOSE to IEDs in this project, but their files are not imported. Each new state they send appears as one dot flying into the IED that receives it.',
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

/** Backend equipment state (Dbpos) in plain words; 0 is a switch between positions. */
const SWITCH_WORDS = ['moving', 'open', 'closed', 'invalid'];
const SWITCHES = new Set(['CB', 'DS', 'ES']);
const STATE_ROLES = new Set(['direct', 'single', 'open', 'closed']);
/** A receiver's short line after a real stNum increase from a sender on the drawing. */
export const GOT_IT = '✓ got it';
const plural = (count: number) => `${count} IED${count === 1 ? '' : 's'}`;

/** What one change did, in plain words (ADR-0103): "open" for a breaker from the backend's own
 * equipment state, "Trip on" for a relay row, else the signal's own words. */
export function changeWhat(group: ChangeGroup, equipment: Record<string, { value: unknown; at: number }> = {}): string {
  const origin = group.origin;
  const last = group.signals.at(-1);
  // Only a new stNum is known (no publisher value seen): say so plainly; the stNum is in the sheet.
  const change = group.lines[0]?.pulse.change || '';
  if (!last) return !change || change.startsWith('stNum') ? 'new state' : change;
  const state = origin ? equipment[origin.nodeId] : undefined;
  if (origin && SWITCHES.has(origin.kind) && STATE_ROLES.has(origin.role) && state && state.at >= group.start - 500
    && typeof state.value === 'number' && SWITCH_WORDS[state.value]) return SWITCH_WORDS[state.value];
  if (origin?.row) return `${origin.row} ${valueWords(last.value)}`.trim();
  const refs = new Set(group.signals.map(signal => signal.ref));
  return refs.size > 1 ? `${refs.size} signals changed` : last.text || 'new state';
}

const capital = (text: string) => text.charAt(0).toUpperCase() + text.slice(1);

/** The one chip at the origin: "Open · sent to 4 IEDs" or "Open · 4 of 4 IEDs got it"; never
 * "received" without a real stNum increase. Its hover says who and why. */
export function changeChip(group: ChangeGroup, equipment: Record<string, { value: unknown; at: number }> = {}) {
  const what = changeWhat(group, equipment);
  const total = group.receivers.length;
  const reach = group.kind === 'observed' ? `${group.got.length} of ${plural(total)} got it` : `sent to ${plural(total)}`;
  const full = group.origin ? `${symbolName(group.origin)} ${what}` : `${group.publisher}: ${what}`;
  const title = [`${capital(full)} (GOOSE from ${group.publisher})`,
    group.kind === 'observed' ? `New state seen on the network for: ${group.got.join(', ')}`
      : `Expected to reach: ${group.receivers.join(', ')}. No GOOSE check has confirmed it yet.`,
    'Click a line for that link, or here for all GOOSE of this IED'].join('\n');
  return { text: `${capital(what)} · ${reach}`, full, title };
}
