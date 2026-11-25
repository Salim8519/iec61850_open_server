import { releaseOf, versionCode } from '../../copy/releaseNotes/index.ts';

export type WhatsNewChoice = 'show' | 'mark' | 'none';
type Seen = { tour_seen_version: number; tour_version: number; whats_new_seen?: number };

/** Whether to open What's new for this version at start (ADR-0148):
 *  - none: no notes for this version, or it was already seen;
 *  - mark: a first start (the tour opens instead), so remember the version quietly;
 *  - show: an update since the window was last closed. */
export function whatsNewChoice(prefs: Seen, current: string): WhatsNewChoice {
  const code = versionCode(current);
  if (code === null || !releaseOf(current)) return 'none';
  if ((prefs.whats_new_seen ?? 0) >= code) return 'none';
  const firstStart = prefs.tour_seen_version < prefs.tour_version && prefs.whats_new_seen === undefined;
  return firstStart ? 'mark' : 'show';
}

/** Splits "Fixed: text" into its kind and the text (capitalised); a note without a kind keeps its words. */
export function noteParts(note: string): { kind: string; text: string } {
  const match = /^(New|Fixed|Changed|Improved|Faster|Removed|Security): (.*)$/s.exec(note);
  if (!match) return { kind: '', text: note };
  return { kind: match[1], text: match[2].charAt(0).toUpperCase() + match[2].slice(1) };  // it starts the line now
}
