import type { CommandOptions } from '../api/types.ts';

/** Command options of a breaker, disconnector or earth switch (ADR-0108). The Go engine checks them
 *  and remembers the last ones sent in the node; the browser only decides whether to send them. */
export const DEFAULT_OPTIONS: CommandOptions = { test: false, interlock_check: false, synchro_check: false, orcat: 0, orident: '' };
export const MAX_ORIDENT = 64;

export const sameOptions = (a: CommandOptions, b: CommandOptions): boolean =>
  a.test === b.test && a.interlock_check === b.interlock_check && a.synchro_check === b.synchro_check
  && a.orcat === b.orcat && a.orident === b.orident;

/** The options a node starts with: what it remembers, else the defaults. */
export const startOptions = (stored: CommandOptions | undefined): CommandOptions => ({ ...DEFAULT_OPTIONS, ...stored });

/** What goes into the request: nothing while a node without remembered options keeps the defaults,
 *  so those requests stay exactly as before; otherwise the options shown. */
export function optionsToSend(stored: CommandOptions | undefined, shown: CommandOptions): CommandOptions | undefined {
  return !stored && sameOptions(shown, DEFAULT_OPTIONS) ? undefined : shown;
}

/** The body of an Open/Close select or operate request. */
export function commandBody(action: string, options: CommandOptions | undefined): { action: string; options?: CommandOptions } {
  return options ? { action, options } : { action };
}

/** How many options differ from the defaults (shown beside the folded title). */
export function changedCount(options: CommandOptions): number {
  return (Object.keys(DEFAULT_OPTIONS) as (keyof CommandOptions)[]).filter(key => options[key] !== DEFAULT_OPTIONS[key]).length;
}
