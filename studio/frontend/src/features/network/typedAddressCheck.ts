import type { Adapter, PlanRow } from '../../api/network';
import { CHECKING_HINT, duplicateHint, otherCardHint } from '../../copy/addressHints.ts';
import type { Manual } from './AddressTable';

/** What the page can say at once about a typed address while the engine's answer is still on its way.
 *  Only a first impression: the engine's answer replaces it and alone decides whether the apply button works. */
export type Provisional = { status: 'duplicate' | 'used_by_other_adapter' | 'checking'; hint: string };

const CHECKING: Provisional = { status: 'checking', hint: CHECKING_HINT };

/** The typed addresses as the engine should read them: spaces around an address are not part of it. */
export function cleanManual(manual: Manual): Manual {
  return Object.fromEntries(Object.entries(manual).map(([id, item]) => [id, { ...item, ip: item.ip.trim() }]));
}

const sameTyped = (left?: Manual[string], right?: Manual[string]) =>
  left !== undefined && right !== undefined && left.ip.trim() === right.ip.trim() && left.prefix === right.prefix;

/** For every row whose typed address changed since the last answer: a duplicate or an address of this PC's other
 *  card, found in memory at once, otherwise "checking". Same order of ownership as the engine: addresses planned
 *  without typing come first, then typed rows from the top of the list. */
export function provisionalChecks(rows: PlanRow[], manual: Manual, answered: Manual | null,
  adapters: Adapter[] | null, guid: string): Record<string, Provisional> {
  const found: Record<string, Provisional> = {};
  const addressOf = (row: PlanRow) => manual[row.ied_id] ? manual[row.ied_id].ip.trim() : row.proposed;
  rows.forEach((row, index) => {
    const typed = manual[row.ied_id];
    if (!typed || sameTyped(typed, answered?.[row.ied_id])) return;
    const ip = typed.ip.trim();
    if (!ip) { found[row.ied_id] = CHECKING; return; }
    const holder = rows.find((other, at) => other !== row && addressOf(other) === ip && (!manual[other.ied_id] || at < index));
    const card = (adapters ?? []).find(item => item.guid.toUpperCase() !== guid.toUpperCase()
      && item.addresses.some(address => address.ip === ip));
    if (holder) {
      found[row.ied_id] = { status: 'duplicate', hint: duplicateHint(holder.name, ip, row.kind === 'snmp' ? 'device' : 'IED') };
    } else if (card) {
      found[row.ied_id] = { status: 'used_by_other_adapter', hint: otherCardHint(ip, card.name) };
    } else found[row.ied_id] = CHECKING;
  });
  return found;
}
