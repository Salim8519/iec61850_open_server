/** The generator form's own check (change 0177), so a wrong value shows one red line under the field and
 *  is not sent (interactions.md). The engine still checks it too, with its own words. */
export const MIN_PERIOD = 0.1;
export const PERIOD_PROBLEM = 'Cycle time must be at least 0.1 seconds.';

/** The words under the cycle time field, or '' when the form may be saved. Every mode but CSV playback
 *  sends the cycle time (the engine checks it in all of them), so all of those are checked. */
export function periodProblem(mode: string, period: number): string {
  if (mode === 'csv') return '';
  return Number.isFinite(period) && period >= MIN_PERIOD ? '' : PERIOD_PROBLEM;
}
