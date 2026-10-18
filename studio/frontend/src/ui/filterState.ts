// The shared filter model of every list (change 0093). Pure functions only: the lists apply
// them to rows the browser already has, and the FilterBar only displays and changes values.

export type Choice = { value: string; label: string };
export type FilterGroup = { key: string; label: string; options: Choice[]; initial: string };
export type FilterToggle = { key: string; label: string; hint?: string; initial?: boolean; remember?: boolean };
export type FilterSelect = { key: string; label: string; all: string };
export type FilterSpec = {
  search?: string; groups?: FilterGroup[]; toggles?: FilterToggle[]; selects?: FilterSelect[];
};
export type FilterValues = Record<string, string | boolean>;
export type Stored = Record<string, string | boolean>;

/** The values a list starts with: empty search, each group's first choice, toggles off. */
export function initialValues(spec: FilterSpec): FilterValues {
  const values: FilterValues = { text: '' };
  for (const group of spec.groups ?? []) values[group.key] = group.initial;
  for (const toggle of spec.toggles ?? []) values[toggle.key] = toggle.initial ?? false;
  for (const select of spec.selects ?? []) values[select.key] = '';
  return values;
}

/** Remembered choices on top of the initial values; unknown keys and invalid choices are ignored. */
export function restore(spec: FilterSpec, stored?: Stored | null): FilterValues {
  const values = initialValues(spec);
  if (!stored) return values;
  for (const group of spec.groups ?? []) {
    const value = stored[group.key];
    if (typeof value === 'string' && group.options.some(option => option.value === value)) values[group.key] = value;
  }
  for (const toggle of spec.toggles ?? []) {
    if (toggle.remember !== false && typeof stored[toggle.key] === 'boolean') values[toggle.key] = stored[toggle.key];
  }
  for (const select of spec.selects ?? []) {
    if (typeof stored[select.key] === 'string') values[select.key] = stored[select.key];
  }
  return values;
}

/** What is saved for next time: chips and dropdowns, never the typed search or a Pause. */
export function rememberable(spec: FilterSpec, values: FilterValues): Stored {
  const stored: Stored = {};
  for (const group of spec.groups ?? []) stored[group.key] = String(values[group.key] ?? group.initial);
  for (const toggle of spec.toggles ?? []) if (toggle.remember !== false) stored[toggle.key] = Boolean(values[toggle.key]);
  for (const select of spec.selects ?? []) stored[select.key] = String(values[select.key] ?? '');
  return stored;
}

/** A remembered dropdown choice that is not in today's list counts as "all". */
export function effective(spec: FilterSpec, values: FilterValues, options: Record<string, Choice[]> = {}): FilterValues {
  const result = { ...values };
  for (const select of spec.selects ?? []) {
    const list = options[select.key] ?? [];
    if (result[select.key] && !list.some(option => option.value === result[select.key])) result[select.key] = '';
  }
  return result;
}

/** True when anything differs from the initial values, so the bar offers Clear. */
export function isFiltered(spec: FilterSpec, values: FilterValues): boolean {
  const start = initialValues(spec);
  return Object.keys(start).some(key => (key === 'text' ? String(values.text ?? '').trim() : values[key]) !== start[key]);
}

/** "12 of 340" while filtered, otherwise the plain total. */
export function countText(shown: number, total: number, filtered: boolean): string {
  return filtered ? `${shown.toLocaleString()} of ${total.toLocaleString()}` : total.toLocaleString();
}

/** Case-insensitive search across any fields; an empty search matches everything. */
export function textMatch(text: unknown, ...fields: unknown[]): boolean {
  const needle = String(text ?? '').trim().toLowerCase();
  if (!needle) return true;
  const haystack = fields.map(field => field === null || field === undefined ? '' : String(field)).join(' ').toLowerCase();
  return needle.split(/\s+/).every(word => haystack.includes(word));
}

/** Dropdown choices from the values in a list, sorted, without duplicates or blanks. */
export function choicesOf(values: (string | null | undefined)[]): Choice[] {
  return [...new Set(values.filter((value): value is string => Boolean(value)))]
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true })).map(value => ({ value, label: value }));
}
