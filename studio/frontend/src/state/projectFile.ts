// ScadaBench project files are .scadabench (ADR-0096). The old .subsim extension is not opened:
// the user is told why in plain words, before anything is sent. The engine refuses it too.

export const PROJECT_EXTENSION = '.scadabench';
export const OLD_PROJECT_EXTENSION = '.subsim';
export const OLD_PROJECT_MESSAGE = 'This is an old project file (.subsim). ScadaBench now uses .scadabench project files.';

/** What a picked file is: a project to open, an old project to refuse, or an SCL file to import. */
export type FileKind = 'project' | 'old-project' | 'scl';

export function fileKind(name: string): FileKind {
  const lower = name.toLowerCase();
  if (lower.endsWith(PROJECT_EXTENSION)) return 'project';
  return lower.endsWith(OLD_PROJECT_EXTENSION) ? 'old-project' : 'scl';
}

/** The file chooser lists SCL files and .scadabench projects only; a dropped .subsim is still explained. */
export const OPEN_FILE_ACCEPT = `.cid,.icd,.scd,.iid,${PROJECT_EXTENSION}`;
