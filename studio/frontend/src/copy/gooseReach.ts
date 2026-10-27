import type { Status } from './status.ts';

/** Words for "Where GOOSE goes" (ADR-0122, as ADR-0073): inside ScadaBench by default; shared with the network
 * only after the user confirms. The Npcap line and the "nobody there" hint stay from ADR-0119. */
export const REACH_COPY = {
  title: 'Where GOOSE goes',
  inside: 'Inside ScadaBench only',
  insideLine: 'Safe default. Simulated IEDs hear each other. Nothing is sent to the network.',
  outside: 'Also share with the network',
  outsideLine: 'Real IEDs, test sets and other PCs on the adapter receive the GOOSE of your simulated IEDs.',
  adapter: 'Network adapter',
  auto: 'Picked automatically',
  chosen: 'Chosen under Advanced',
  protection: 'Protection, always on',
  guard: ['Our own echoes dropped', 'Network GOOSE never resent', 'Held if a real device uses the same name', 'Rate limit per IED'],
  alone: (adapter: string) => `Nothing else on ${adapter} has sent GOOSE yet, and a GOOSE your project expects is still missing. If the other PC or VM is running, check that it is on the same network.`,
  review: 'Review and confirm',
  advanced: 'Advanced: choose the adapter',
  automatic: 'Automatic (recommended)',
  lockedWhileSharing: 'Switch sharing off to change the adapter.',
  getNpcap: 'Download Npcap (free)',
  pill: 'GOOSE is shared with the network',
  pillHint: 'The GOOSE of your simulated IEDs also goes to the network. Click to change.',
};

export type Risk = { level: 'high' | 'warning'; reasons: string[] };

/** The share warning; stronger words when real devices are likely to act on the GOOSE (GOOSE-25). */
export function shareWarning(risk: Risk, adapter: string) {
  const high = risk.level === 'high';
  return {
    title: high ? `High risk: real devices on “${adapter}” may act on this GOOSE`
      : `Share GOOSE on “${adapter}”?`,
    body: high
      ? 'Real IEDs can trip, block or interlock on GOOSE from your simulated IEDs. Only continue on a test network with no live equipment.'
      : 'Everything plugged into this adapter will receive the GOOSE of your simulated IEDs. Use a test network.',
    guard: 'The smart guard stays on: it listens for 3 seconds first, never repeats network GOOSE, and stops an IED and asks you if a real device sends the same GOOSE.',
    confirm: high ? 'I understand, share anyway' : 'Share GOOSE',
    cancel: 'Cancel',
  };
}

export const CONFLICT_COPY = {
  title: (ied: string) => `A real device sends the same GOOSE as ${ied}`,
  body: (mac: string, ref: string) => `Device ${mac} on the network sends ${ref}. ScadaBench stopped sending this IED's GOOSE outside so the two do not fight. Inside ScadaBench it keeps working.`,
  stop: 'Keep it stopped outside',
  continue: 'Continue anyway',
  sure: 'Two senders of the same GOOSE can confuse or trip real equipment. Keep sending?',
  yes: 'Yes, keep sending',
};

/** One status pill for the reach state the Go engine reports (ADR-0122). */
export const REACH_STATES: Record<string, Status> = {
  inside: { label: 'Inside ScadaBench', tone: 'good', hint: 'GOOSE runs inside ScadaBench. Nothing is sent to the network.' },
  listening: { label: 'Inside · listening', tone: 'good', hint: 'GOOSE runs inside ScadaBench and real GOOSE on the adapter is heard. Nothing is sent.' },
  checking: { label: 'Checking network', tone: 'waiting', hint: 'ScadaBench listens for 3 seconds for real devices before sending GOOSE.' },
  share_at_run: { label: 'Shares at Run', tone: 'warning', hint: 'Sharing is on. GOOSE goes to the network after the 3-second check once you press Run.' },
  sharing: { label: 'Shared with network', tone: 'warning', hint: 'Simulated GOOSE also goes to the network. The smart guard is on.' },
  confirm: { label: 'Waiting for you', tone: 'warning', hint: 'Real devices are on this network. Confirm before ScadaBench sends GOOSE next to them.' },
  share_held: { label: 'Sharing paused', tone: 'problem', hint: 'Sharing is switched on but cannot run right now. See the reason.' },
  nothing_to_watch: { label: 'No GOOSE to watch', tone: 'off', hint: 'No running IED sends or expects GOOSE in this project.' },
  failed: { label: 'GOOSE stopped', tone: 'problem', hint: 'The GOOSE engine stopped. See Activity for the reason, then press Run again.' },
  off: { label: 'Not running', tone: 'off', hint: 'Press Run to start GOOSE.' },
  stopped: { label: 'Not running', tone: 'off', hint: 'Press Run to start GOOSE.' },
};

export const reachStatus = (state: string): Status => REACH_STATES[state] ?? REACH_STATES.inside;
