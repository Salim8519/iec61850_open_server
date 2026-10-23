import type { GooseLeaf, GooseMessage } from '../api/gooseMessages';
import type { EquipmentKind, GooseLinkSetting, Ied } from '../api/types';
import { positionWord } from './pointDisplay.ts';

/** What the GOOSE section of an equipment sheet offers (ADR-0112). Go checks every saved link and
 * every send again; these helpers only keep the choices short and the buttons right. Pure. */
const NO_GOOSE_LINK: readonly EquipmentKind[] = ['JUNCTION', 'INFEED', 'POINT_PANEL', 'IED'];

export const takesGooseLink = (kind: EquipmentKind) => !NO_GOOSE_LINK.includes(kind);
export const isSwitch = (kind: EquipmentKind) => kind === 'CB' || kind === 'DS' || kind === 'ES';
/** "PUBCTRL/XCBR1.Pos.stVal" -> "XCBR1.Pos.stVal". */
export const leafName = (ref: string) => ref.split('/').pop() || ref;

/** Simulated IEDs whose files send GOOSE: the only ones the drawing can send for. */
export function gooseSenders(ieds: Ied[]): Ied[] {
  return ieds.filter(ied => ied.role === 'simulated' && (ied.goose?.publishes?.length ?? 0) > 0);
}

export function sendableLeaves(message: GooseMessage): GooseLeaf[] {
  return message.members.flatMap(member => member.refs).filter(leaf => leaf.sendable && !!leaf.value_type);
}

/** Values the link may name: two on/off values for "contacts"; a switch never sends a number. */
export function linkChoices(message: GooseMessage, kind: EquipmentKind, mode: GooseLinkSetting['mode']): GooseLeaf[] {
  const leaves = sendableLeaves(message);
  if (mode === 'contacts') return leaves.filter(leaf => leaf.value_type === 'onoff');
  return isSwitch(kind) ? leaves.filter(leaf => leaf.value_type !== 'number') : leaves;
}

export type LinkDraft = Partial<GooseLinkSetting> & { mode: GooseLinkSetting['mode'] };

/** Only a whole link is saved: IED, message and its value(s). */
export function linkComplete(link: LinkDraft): link is GooseLinkSetting {
  if (!link.ied_id || !link.gocb_ref) return false;
  return link.mode === 'contacts' ? !!link.open && !!link.closed && link.open !== link.closed && !link.one : !!link.one;
}

/** The stored shape of a whole link (the unused value fields are null). */
export function storedLink(link: GooseLinkSetting): GooseLinkSetting {
  const contacts = link.mode === 'contacts';
  return { ied_id: link.ied_id, gocb_ref: link.gocb_ref, mode: link.mode, one: contacts ? null : link.one,
    open: contacts ? link.open : null, closed: contacts ? link.closed : null };
}

export type SendAction = 'open' | 'closed' | 'on' | 'off';
/** The buttons of a saved link: Open/Closed for positions and switches, On/Off for an on/off value
 * of other equipment, or a number box ('number'). null until the value's type is known. */
export function linkActions(kind: EquipmentKind, link: GooseLinkSetting, leaf?: GooseLeaf): SendAction[] | 'number' | null {
  if (link.mode === 'contacts') return ['open', 'closed'];
  if (!leaf?.value_type) return null;
  if (leaf.value_type === 'number') return 'number';
  return leaf.value_type === 'position' || isSwitch(kind) ? ['open', 'closed'] : ['on', 'off'];
}

export const ACTION_LABEL: Record<SendAction, string> = { open: 'Open', closed: 'Closed', on: 'On', off: 'Off' };

/** The value a message holds now, in words ("" when unknown). */
export function valueWords(leaf: Pick<GooseLeaf, 'value_type' | 'value'>): string {
  const { value } = leaf;
  if (value === null || value === undefined) return '';
  if (leaf.value_type === 'onoff' && typeof value === 'boolean') return value ? 'On' : 'Off';
  if (leaf.value_type === 'position' && typeof value === 'number') return positionWord(value);
  return typeof value === 'object' ? '' : String(value);
}

/** The leaf a link names, from the IED's message answer. */
export function leafOf(messages: GooseMessage[] | undefined, gocbRef: string, ref: string | null): GooseLeaf | undefined {
  const message = messages?.find(item => item.gocb_ref === gocbRef);
  return ref ? message?.members.flatMap(member => member.refs).find(leaf => leaf.ref === ref) : undefined;
}
