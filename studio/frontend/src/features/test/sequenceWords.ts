// Plain words for timed sequences: the step table and the run state never show internal names.

const ACTIONS: Record<string, string> = { set: 'Set value', assert: 'Check value' };

/** "Set value" / "Check value", the same words as the Action list above the table. */
export function stepAction(action: string) {
  return ACTIONS[action] ?? action;
}

/** "Ready", "Running", "Finished", "Stopped" or "Failed at step 4 of 4" for one sequence. */
export function runWords(state: string | undefined, completed: number | undefined, total: number) {
  if (!state) return 'Ready';
  if (state === 'failed') {
    const step = Math.min((completed ?? 0) + 1, total);
    return total ? `Failed at step ${step} of ${total}` : 'Failed';
  }
  const word: Record<string, string> = { running: 'Running', completed: 'Finished', cancelled: 'Stopped',
    ready: 'Ready' };
  return word[state] ?? state.charAt(0).toUpperCase() + state.slice(1);
}

export const stepCount = (count: number) => `${count} ${count === 1 ? 'step' : 'steps'}`;
