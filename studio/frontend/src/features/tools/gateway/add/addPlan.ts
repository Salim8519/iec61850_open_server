// Display helpers of the Add points dialog (ADR-0105). Pure: the engine plans and checks every point.
import type { Candidate, Pick, Plan } from '../../../../api/gatewayAdd';
import type { Ied } from '../../../../api/types';

/** The type list in two groups: most used first (catalog order), the rest under "More types". */
export function splitTypes<T extends { most_used: boolean }>(types: T[]): { most: T[]; more: T[] } {
  return { most: types.filter(entry => entry.most_used), more: types.filter(entry => !entry.most_used) };
}

/** IEDs whose name matches every typed word (any case). */
export function filterIeds(ieds: Ied[], query: string): Ied[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  return ieds.filter(ied => words.every(word => ied.name.toLowerCase().includes(word)));
}

/** Ticks a reference on or off, keeping the order in which they were ticked. */
export function toggle(ticked: string[], ref: string): string[] {
  return ticked.includes(ref) ? ticked.filter(item => item !== ref) : [...ticked, ref];
}

/** Ticks every shown candidate that is not in the list yet. */
export function tickShown(ticked: string[], shown: Candidate[]): string[] {
  const more = shown.filter(item => !item.taken && !ticked.includes(item.ref)).map(item => item.ref);
  return [...ticked, ...more];
}

export type RowEdit = { name?: string; address?: number };

/** The ticked signals as the engine wants them, with the names and addresses typed in the rows. */
export function picksOf(ticked: string[], edits: Record<string, RowEdit>): Pick[] {
  return ticked.map(ref => ({ ref, ...(edits[ref]?.name !== undefined ? { name: edits[ref].name } : {}),
    ...(edits[ref]?.address !== undefined ? { address: edits[ref].address } : {}) }));
}

/** Whether "Add N points" may be pressed: every row right and, for a real device, "I understand" ticked. */
export function canAdd(plan: Plan | null, real: boolean, understood: boolean): boolean {
  return !!plan && plan.ok && plan.points.length > 0 && (!real || understood);
}

/** A typed address: digits only, as the point list reads it ("1 042" is 1042); empty means "next free". */
export function typedAddress(text: string): number | undefined {
  const digits = text.replace(/\D/g, '');
  return digits === '' ? undefined : Number(digits);
}

/** The problems of a plan, one line per bad row. */
export const problemLines = (plan: Plan | null): string[] => (plan?.rows ?? []).filter(row => row.problem).map(row => row.problem);

/** Whether new points belong to a real device (those need the warning and "I understand"). */
export const hasRealDevice = (points: { ied_role?: string }[]) => points.some(point => point.ied_role === 'real');
