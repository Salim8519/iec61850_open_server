import type { Diagram } from '../../api/types';

export type SldSource = { id: string; filename: string; ieds: string[]; substations: string[]; equipment: number };
export type SldMode = 'beside' | 'replace';
export type SldPreview = {
  diagram: Diagram; revision: number; layout: 'esld' | 'automatic'; removed: number;
  drawn: { path: string; kind: string; label: string; bound: boolean }[];
  skipped: { path: string; type: string; reason: string }[];
  unbound: { path: string; kind: string; reasons: string[] }[];
  diagnostics: string[]; already_present: string[];
};

export const NO_LAYOUT_HINT = 'This file has no substation layout. Ask for the SCD from the system integrator.';

/** Only files with a Substation section can be drawn; otherwise say why in plain words. */
export function sldAvailability(sources: SldSource[]): { enabled: boolean; hint: string; usable: SldSource[] } {
  const usable = sources.filter(source => source.substations.length > 0);
  if (!sources.length) return { enabled: false, hint: 'Import the substation SCD file first.', usable };
  return usable.length ? { enabled: true, hint: '', usable } : { enabled: false, hint: NO_LAYOUT_HINT, usable };
}

/** Counts first, details after: what the preview will do to the diagram. */
export function sldSummary(preview: SldPreview): string[] {
  const bound = preview.drawn.filter(item => item.bound).length;
  const lines = [`${preview.drawn.length} items will be drawn, ${bound} of them linked to an IED.`];
  if (preview.skipped.length) lines.push(`${preview.skipped.length} items are not drawn and are listed below with the reason.`);
  if (preview.already_present.length) lines.push(`${preview.already_present.length} items are already on the diagram and are kept as they are.`);
  if (preview.removed) lines.push(`Replace removes the ${preview.removed} items that are on the diagram now.`);
  lines.push(preview.layout === 'esld' ? 'Positions come from the drawing in the SCD file.' : 'Positions were arranged automatically: voltage levels as rows, bays as columns.');
  return lines;
}

/** SCL types that have a symbol in the Add menu but that Build from SCD does not place yet. */
const ADD_BY_HAND: Record<string, string> = { PTR: 'a Tap-changer transformer',
  PowerTransformer: 'a Tap-changer transformer', IFL: 'an Infeed' };

/** Why one item is not drawn. The engine's reason (shared word for word with the Python engine) says
 *  "no symbol yet"; for a type the Add menu has, that is wrong, so say where to find it (change 0176). */
export function skippedReason(item: { type: string; reason: string }): string {
  const symbol = ADD_BY_HAND[item.type];
  return symbol ? `Build from SCD does not place this type yet. Add ${symbol} from the Add menu.` : item.reason;
}

/** A preview made for an older project revision must be made again before applying. */
export function previewIsCurrent(preview: SldPreview, revision: number): boolean {
  return preview.revision === revision;
}
