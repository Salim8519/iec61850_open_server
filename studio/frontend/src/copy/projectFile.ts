// Words of the File menu, its New/Close dialog and the Undo/Redo buttons (ADR-0135).
export const PROJECT_FILE_COPY = {
  menu: { open: 'Open file…', new: 'New project…', save: 'Save project', saveAs: 'Save as…', close: 'Close project…' },
  newTitle: 'Start a new project',
  newIntro: (name: string) => `“${name}” closes and an empty project opens. ScadaBench keeps one project open at a time.`,
  closeTitle: (name: string) => `Close “${name}”?`,
  closeIntro: 'An empty project opens in its place. Open file brings any saved project back.',
  nameLabel: 'Name of the new project', namePlaceholder: 'Untitled substation',
  copyHint: 'Want to keep this project? Save a copy first. It is gone from ScadaBench once it closes.',
  saveCopy: 'Save a copy…', copySaved: 'Copy saved.', copyDownloaded: 'Copy saved to your Downloads folder.',
  cancel: 'Cancel', newAction: 'Start new project', closeAction: 'Close project', working: 'Working…',
  saved: (how: 'saved' | 'downloaded') => how === 'saved' ? 'Project saved as a file.' : 'Project saved to your Downloads folder.',
  runningHint: 'Stop the IEDs first.',
};

export const UNDO_COPY = {
  undo: 'Undo', redo: 'Redo',
  undoTip: (what: string) => `Undo ${what} (Ctrl+Z)`,
  redoTip: (what: string) => `Redo ${what} (Ctrl+Y)`,
  nothingUndo: 'Nothing to undo', nothingRedo: 'Nothing to redo',
  running: 'Stop the IEDs to undo or redo. Changes made while they run are not undone.',
};

/** File › auto save (ADR-0139). On (default) saves every change at once; off keeps changes in the
 *  open project until Save project or Ctrl+S, and the next start opens the last saved project. */
export const AUTOSAVE_COPY = {
  turnOff: 'Turn off auto save', turnOn: 'Turn on auto save',
  turningOff: 'Turning auto save off…', turningOn: 'Saving and turning auto save on…',
  off: 'Auto save off', unsaved: 'Not saved · Ctrl+S',
  offTip: 'Auto save is off. Changes are kept while ScadaBench is open; File › Save project (Ctrl+S) writes them.',
  unsavedTip: 'Auto save is off and there are changes that are not saved. Press Ctrl+S or File › Save project. ' +
    'If ScadaBench closes first, the next start opens the last saved project.',
};

/** The words beside the File menu: the diagram's own state, or with auto save off whether the
 *  project has changes that Save has not written yet. */
export function saveWords(saveState: string, auto: { on: boolean; unsaved: boolean }): { text: string; tip?: string } {
  if (auto.on || saveState === 'Not saved') return { text: saveState };
  return auto.unsaved || saveState === 'Saving...'
    ? { text: AUTOSAVE_COPY.unsaved, tip: AUTOSAVE_COPY.unsavedTip } : { text: AUTOSAVE_COPY.off, tip: AUTOSAVE_COPY.offTip };
}
