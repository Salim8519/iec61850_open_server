/** Remembers where each signal picker was (IED, search, report, page) for this browser tab,
 *  so closing and reopening a picker, or picking a wrong point, never starts the search over. */
export type PickerView = { iedId: string; query: string; reportRef: string; offset: number; pointKind?: string;
  /** The type the slot opened with when pointKind was chosen: another slot under the same name starts on its own. */
  kindFor?: string;
  /** Object (logical node) and the quality/time switch of the smart picker (ADR-0113). */
  ln?: string; helpers?: boolean };

const views = new Map<string, PickerView>();

export const EMPTY_VIEW: PickerView = { iedId: '', query: '', reportRef: '__all__', offset: 0 };

/** The view to open with: the remembered one, unless the current binding lives on another IED. */
export function initialView(key: string | undefined, bindingIedId?: string, fallbackIedId = ''): PickerView {
  const saved = key ? views.get(key) : undefined;
  if (bindingIedId && saved?.iedId !== bindingIedId) return { ...EMPTY_VIEW, iedId: bindingIedId };
  return saved ?? { ...EMPTY_VIEW, iedId: fallbackIedId };
}

export function rememberView(key: string | undefined, view: PickerView) {
  if (key) views.set(key, view);
}
