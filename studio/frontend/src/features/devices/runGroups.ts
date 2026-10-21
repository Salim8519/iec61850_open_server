import type { Ied } from '../../api/types';
import type { RunSummary } from '../../copy/runShape';

/** The list order while some IEDs run: running ones first, each group keeping its own order.
 *  Returns no labels when nothing or everything runs, so the usual list stays unchanged. */
export function runGroups(shown: Ied[], summary: RunSummary): { label: string | null; ieds: Ied[] }[] {
  if (summary.shape !== 'partial') return [{ label: null, ieds: shown }];
  const live = new Set(summary.running.map(ied => ied.id));
  const running = shown.filter(ied => live.has(ied.id));
  const rest = shown.filter(ied => !live.has(ied.id));
  return [{ label: `Running · ${running.length}`, ieds: running }, { label: `Not running · ${rest.length}`, ieds: rest }]
    .filter(group => group.ieds.length);
}
