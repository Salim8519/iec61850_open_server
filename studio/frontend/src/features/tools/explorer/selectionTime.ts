export type Selection = { label: string; value: unknown; until: number };

/** Seconds left of a selection, at least 0 (rounded up so "1 s" shows until the very end). */
export const secondsLeft = (selection: Selection, now: number) => Math.max(0, Math.ceil((selection.until - now) / 1000));
