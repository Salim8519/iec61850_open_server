/** One on/off app setting that is saved in the data folder (change 0176). The switch moves at
 *  once; a save that fails moves it back and keeps the reason, so it never looks saved when it
 *  is not. */
export type SavedSwitch = {
  get: () => boolean;
  /** Why the last change was not saved ('' when it was). */
  error: () => string;
  subscribe: (listener: () => void) => () => void;
  /** Takes the saved value without saving it again. */
  load: (value: boolean) => void;
  set: (next: boolean) => Promise<void>;
};

const messageOf = (error: unknown) => error instanceof Error ? error.message : String(error);

export function createSavedSwitch(save: (next: boolean) => Promise<unknown>): SavedSwitch {
  let value = false;
  let error = '';
  const listeners = new Set<() => void>();
  const notify = () => listeners.forEach(listener => listener());
  return {
    get: () => value,
    error: () => error,
    subscribe: listener => { listeners.add(listener); return () => { listeners.delete(listener); }; },
    load: next => { value = next; notify(); },
    set: async next => {
      const previous = value;
      value = next;
      error = '';
      notify();
      try { await save(next); } catch (failure) {
        if (value === next) value = previous;
        error = messageOf(failure);
        notify();
      }
    },
  };
}
