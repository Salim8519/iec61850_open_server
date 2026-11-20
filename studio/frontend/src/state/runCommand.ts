import { createContext, useContext } from 'react';

/** Runs a task with the window's busy state and error line, like `perform`. */
export type Command = (task: () => Promise<unknown>, label?: string) => Promise<void>;

/** Runtime-only actions (change 0224, trial U1): set a value, pulse, resume a generator, the SLD Test
 *  buttons, a GOOSE send and starting a sequence change live values only, never the project
 *  (`studio/tools/trials/u1_revision_check.mjs` proves it on the engine). They show busy and errors
 *  as every action does, without reading the whole project again; their results arrive as live
 *  events. Operate stays a `perform`: the engine remembers the last position and options in the
 *  project. The provider is App (`useStudio().command`). */
export const CommandContext = createContext<Command | null>(null);

/** The app's runtime-only runner, or `fallback` (a page's `perform`) outside the app's provider. */
export function useCommand(fallback: Command): Command {
  return useContext(CommandContext) ?? fallback;
}
