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
