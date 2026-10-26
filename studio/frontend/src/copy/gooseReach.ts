import type { Status } from './status.ts';

/** Words for "Where GOOSE goes" (ADR-0119): with the IEDs' network by default, or only inside ScadaBench. */
export const REACH_COPY = {
  title: 'Where GOOSE goes',
  auto: 'Same network as your IEDs',
  autoLine: 'Recommended. GOOSE goes where your IEDs are: to the lab network, or only inside ScadaBench when the IEDs use this PC only.',
  local: 'Inside ScadaBench only',
  localLine: 'Simulated IEDs hear each other. Nothing is sent to the network.',
  goesOut: 'GOOSE goes out on',
  listens: 'Listens on',
  picked: 'Picked from the IED network. Only PCs and VMs on this network hear it.',
  chosen: 'Chosen under Advanced. Only PCs and VMs on this network hear it.',
  protection: 'Protection, always on',
  guard: ['Our own echoes dropped', 'Network GOOSE never resent', 'Held if a real device uses the same name', 'Rate limit per IED'],
  alone: (adapter: string) => `Nothing else on ${adapter} has sent GOOSE yet, and a GOOSE your project expects is still missing. If the other PC or VM is running, check that it is on the same network.`,
  review: 'Review and confirm',
  advanced: 'Advanced: choose the adapter',
  adapter: 'Network adapter',
  automatic: 'Automatic (recommended)',
  getNpcap: 'Download Npcap (free)',
  pill: 'GOOSE goes to the network',
  pillHint: 'The GOOSE of your simulated IEDs goes to the IED network. Click to change.',
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
    confirm: high ? 'I understand, send GOOSE' : 'Send GOOSE',
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

/** One status pill for the reach state the Go engine reports (ADR-0119). */
export const REACH_STATES: Record<string, Status> = {
  inside: { label: 'Inside ScadaBench', tone: 'good', hint: 'Your IEDs use this PC only, so GOOSE stays inside ScadaBench.' },
  local: { label: 'Inside ScadaBench', tone: 'good', hint: 'GOOSE stays inside ScadaBench, as you chose. Nothing is sent to the network.' },
  listening: { label: 'Inside · listening', tone: 'good', hint: 'GOOSE stays inside ScadaBench and real GOOSE on the adapter is heard. Nothing is sent.' },
  checking: { label: 'Checking network', tone: 'waiting', hint: 'ScadaBench listens for 3 seconds for real devices before sending GOOSE.' },
  goes_at_run: { label: 'Goes out at Run', tone: 'waiting', hint: 'GOOSE goes to the IED network after the 3-second check once you press Run.' },
  sharing: { label: 'On the IED network', tone: 'warning', hint: 'Simulated GOOSE goes to the IED network. Protection is on.' },
  confirm: { label: 'Waiting for you', tone: 'warning', hint: 'Real devices are on this network. Confirm before ScadaBench sends GOOSE next to them.' },
  share_held: { label: 'Not sent', tone: 'problem', hint: 'GOOSE should go to the IED network but cannot right now. See the reason.' },
  nothing_to_watch: { label: 'No GOOSE to watch', tone: 'off', hint: 'No running IED sends or expects GOOSE in this project.' },
  failed: { label: 'GOOSE stopped', tone: 'problem', hint: 'The GOOSE engine stopped. See Activity for the reason, then press Run again.' },
  off: { label: 'Not running', tone: 'off', hint: 'Press Run to start GOOSE.' },
  stopped: { label: 'Not running', tone: 'off', hint: 'Press Run to start GOOSE.' },
};

export const reachStatus = (state: string): Status => REACH_STATES[state] ?? REACH_STATES.inside;
