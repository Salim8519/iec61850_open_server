import { useCallback, useEffect, useState } from 'react';
import type { Project } from '../api/types';
import { getAutoSave, setAutoSave, type AutoSaveState } from '../api/autoSave';
import { AUTOSAVE_COPY } from '../copy/projectFile';

export type AutoSaveSwitch = AutoSaveState & { toggle: () => void };

/** The engine's auto save switch and whether the open project has unsaved changes. It is read
 *  again whenever the project is read again, so a change or a Save shows at once. */
export function useAutoSaveSwitch(project: Project | null,
  perform: (task: () => Promise<unknown>, label?: string) => Promise<void>): AutoSaveSwitch {
  const [state, setState] = useState<AutoSaveState>({ on: true, unsaved: false });
  const reload = useCallback(() => { getAutoSave().then(setState).catch(() => {}); }, []);
  useEffect(() => { if (project) reload(); }, [project, reload]);
  const toggle = useCallback(() => {
    perform(async () => setState(await setAutoSave(!state.on)), state.on ? AUTOSAVE_COPY.turningOff : AUTOSAVE_COPY.turningOn);
  }, [perform, state.on]);
  return { ...state, toggle };
}
