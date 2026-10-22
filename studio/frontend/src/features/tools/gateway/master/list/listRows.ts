// Display helpers of the master point list (ADR-0110). Pure: the engine compares, plans and checks;
// these only filter, count and word what it answered.
import type { MasterCommand } from '../../../../../api/master.ts';
import type { ComparedRow, Comparison, ExtraRow, ListPlan, ListPoint, MasterType } from '../../../../../api/masterList.ts';
import { STATE_WORDS } from '../../../../../copy/masterList.ts';
import { kindOfMasterCommand, kindOfMasterGroup, type Kind } from '../../../../../points/pointKind.ts';
import { textMatch, type FilterSpec, type FilterValues } from '../../../../../ui/filterState.ts';
import { kindGroup, kindMatches } from '../../../../../ui/kindFilter.ts';

export const ALL = 'all';
export const EXTRA = 'extra';
const STATES = ['received', 'missing', 'wrong_type', 'waiting', 'command'] as const;

/** The kind badge of a master type: what it receives, else what it sends. */
export const kindOfListType = (type: Pick<MasterType, 'group' | 'command'> | undefined): Kind =>
  !type ? 'UN' : type.group ? kindOfMasterGroup(type.group) : kindOfMasterCommand(type.command);

/** The filter bar: search, one chip per result (with its count) plus "Not in your list", and the type dropdown. */
export const LIST_FILTER: FilterSpec = {
  search: 'Search name, address or value',
  groups: [{ key: 'state', label: 'Result', initial: ALL, options: [{ value: ALL, label: 'All' },
    ...STATES.map(state => ({ value: state, label: `${STATE_WORDS[state].symbol} ${STATE_WORDS[state].label}` })),
    { value: EXTRA, label: '+ Not in your list' }] }, kindGroup()],
};

/** The chip counts: every result of the comparison, and the points not in the list. */
export function stateCounts(comparison: Comparison | null): Record<string, Record<string, number>> {
  const counts: Record<string, number> = { [ALL]: comparison?.counts.total ?? 0, [EXTRA]: comparison?.not_in_list.length ?? 0 };
  for (const state of STATES) counts[state] = comparison?.counts[state] ?? 0;
  return { state: counts };
}

/** A listed row passes the search, the result chip and the type dropdown. */
export function listRowMatches(row: ComparedRow, values: FilterValues, type: MasterType | undefined): boolean {
  if (values.state && values.state !== ALL && values.state !== row.state) return false;
  const kind = kindOfListType(type);
  if (!kindMatches(values.kind, kind)) return false;
  return textMatch(values.text, row.point.name, row.point.address, type?.name, type?.type_id, kind, row.value_text, row.engineering);
}

/** A "Not in your list" row passes the search and the type dropdown. */
export function extraMatches(row: ExtraRow, values: FilterValues): boolean {
  const kind = kindOfMasterGroup(row.group);
  return kindMatches(values.kind, kind) && textMatch(values.text, row.label, row.address, row.group_name, row.type, kind, row.value_text);
}

/** The command a Send button opens, as the engine built it from the saved point. */
export const draftOf = (row: ComparedRow): MasterCommand | null => row.command ? { ...row.command } : null;

/** The list with new points added at the end (ids from the engine's plan). */
export const appendPoints = (list: ListPoint[], added: ListPoint[]): ListPoint[] => [...list, ...added];

/** The list without the points whose ids are given. */
export const removePoints = (list: ListPoint[], ids: Set<string>): ListPoint[] => list.filter(point => !ids.has(point.id));

/** The list with one point changed (name, scale, unit …); the engine checks it again when it is saved. */
export const changePoint = (list: ListPoint[], id: string, change: Partial<ListPoint>): ListPoint[] =>
  list.map(point => point.id === id ? { ...point, ...change } : point);

/** Whether "Add N points" may be pressed: there are rows and the engine says every one is right. */
export const planReady = (plan: ListPlan | null) => !!plan && plan.ok && plan.points.length > 0;

/** The problems of a plan, one line per bad row. */
export const planProblems = (plan: ListPlan | null): string[] => (plan?.rows ?? []).filter(row => row.problem).map(row => row.problem);

/** A typed number for scale or offset ("0,1" reads as 0.1); undefined when it is not a number. */
export function typedNumber(text: string): number | undefined {
  const value = Number(text.trim().replace(',', '.'));
  return text.trim() !== '' && Number.isFinite(value) ? value : undefined;
}

/** The result in words with its symbol, never colour alone. */
export const stateText = (row: ComparedRow) => `${STATE_WORDS[row.state].symbol} ${STATE_WORDS[row.state].label}`;

/** The IED check in words: ✓ match, ✗ with the reason, or – when there is nothing to compare yet. */
export const checkText = (row: ComparedRow) => !row.check ? '' : row.check.ok === true ? `✓ ${row.check.text}`
  : row.check.ok === false ? `✗ ${row.check.text}` : `– ${row.check.text}`;
