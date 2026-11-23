import type { CommandControl } from '../../equipment/pointRequests.ts';

// Finding a real IED's commands in Devices (ADR-0147): display only, the engine checks every send.

/** Whether a command matches the search: every typed word is in its reference or class. */
export function matchesSearch(control: Pick<CommandControl, 'ref' | 'cdc'>, query: string): boolean {
  const text = `${control.ref} ${control.cdc}`.toLowerCase();
  return query.toLowerCase().split(/\s+/).filter(Boolean).every(word => text.includes(word));
}

/** The command object of a Data points row (LD/LN.DO of `.Oper.ctlVal`), or null for other rows. */
export function commandRefOf(signalRef: string): string | null {
  const found = /^(.*)\.Oper\.ctlVal(\.f)?$/.exec(signalRef);
  return found ? found[1] : null;
}

