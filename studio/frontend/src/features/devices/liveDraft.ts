import type { LiveSettings } from '../../api/types';

/** When the connection form starts again from the saved settings (fix plan A17): only for another
 *  IED or when the saved values really change. Every project read brings a new `live` object with
 *  the same values; that must not throw away what the user is typing. */
export function draftKey(iedId: string, live: LiveSettings): string {
  return `${iedId}\n${JSON.stringify(live)}`;
}
