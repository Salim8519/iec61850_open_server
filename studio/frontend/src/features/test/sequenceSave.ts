// Saving timed sequences as they change ("save always"): the rules the engine checks (a name of 1 to 100
// characters, 1 to 1000 steps) in plain words, and how the one edited sequence joins the saved list.
import type { Scenario } from '../../api/types.ts';

export type SequenceProblem = { field: 'name' | 'steps'; text: string };

/** What stops this sequence from being saved, or null when it is saved as it is. A new sequence without
 *  steps is not a problem, only not saved yet (`saved` false): the page says so quietly. */
export function sequenceProblem(draft: Scenario, saved: boolean): SequenceProblem | null {
  const name = draft.name.trim();
  if (name === '' || name.length > 100) return { field: 'name', text: 'Give the sequence a name of 1 to 100 characters. It is not saved until it has one.' };
  if (draft.steps.length > 1000) return { field: 'steps', text: 'A sequence can have at most 1000 steps. Remove some steps to save it.' };
  if (draft.steps.length === 0 && saved) {
    return { field: 'steps', text: 'A sequence needs at least one step, so this change is not saved. Add a step, or delete the sequence above.' };
  }
  return null;
}

/** The quiet hint of a new sequence that has no step yet. */
export const NEW_SEQUENCE_HINT = 'A new sequence is saved by itself as soon as it has one step.';

/** Whether this sequence is sent to be saved: it is valid and it is not an empty new one. */
export const canSave = (draft: Scenario, saved: boolean) => !sequenceProblem(draft, saved) && draft.steps.length > 0;

/** The saved list with the edited sequence in its own place (a new one goes to the end). */
export function withSequence(scenarios: Scenario[], draft: Scenario): Scenario[] {
  return scenarios.some(item => item.id === draft.id)
    ? scenarios.map(item => item.id === draft.id ? draft : item)
    : [...scenarios, draft];
}
