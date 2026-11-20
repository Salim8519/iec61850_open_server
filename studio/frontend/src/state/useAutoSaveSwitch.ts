import { useCallback, useEffect, useState } from 'react';
import type { Project } from '../api/types';
import { setAutoSave, type AutoSaveState } from '../api/autoSave';
import { liveFeed } from '../api/polls';
import { useAskAgain } from '../api/useAskAgain';
import { AUTOSAVE_COPY } from '../copy/projectFile';

export type AutoSaveSwitch = AutoSaveState & { toggle: () => void };

export const AUTOSAVE = '/project/autosave';

/** The engine's auto save switch and whether the open project has unsaved changes. The Go engine
 *  pushes it whenever it changes, also after a Save and from another tab (topic project.autosave,
 *  change 0224); a polled engine is asked again whenever the project is read again. */
export function useAutoSaveSwitch(project: Project | null,
  perform: (task: () => Promise<unknown>, label?: string) => Promise<void>): AutoSaveSwitch {
  const [state, setState] = useState<AutoSaveState>({ on: true, unsaved: false });
  const loaded = !!project;
  useEffect(() => {
    if (!loaded) return undefined;
    return liveFeed.subscribe<AutoSaveState>(AUTOSAVE, null, setState, () => {});  // no switch on old engines
  }, [loaded]);
  useAskAgain(loaded ? AUTOSAVE : null, project);
  const toggle = useCallback(() => {
    perform(async () => setState(await setAutoSave(!state.on)), state.on ? AUTOSAVE_COPY.turningOff : AUTOSAVE_COPY.turningOn);
  }, [perform, state.on]);
  return { ...state, toggle };
}
