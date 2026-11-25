// Words of Re-import CID (ADR-0150). Plain English for people new to IEC 61850.
type Counts = { before: number; after: number; added: number; removed: number };

const many = (count: number, word: string) => `${count.toLocaleString()} ${word}${count === 1 ? '' : 's'}`;

export const REIMPORT_COPY = {
  title: (name: string) => `Re-import the CID of ${name}`,
  intro: 'Use a newer file of the same IED. ScadaBench keeps this IED, its settings (role, addresses, Allow commands, '
    + 'network card) and every link to the diagram, panels, sources and scenarios that the new file still has. '
    + 'You see what changes before anything is saved.',
  choose: 'Choose the new CID file…',
  another: 'Choose another file…',
  checking: 'Comparing the new file with this IED…',
  applying: 'Re-importing…',
  apply: (lost: number) => lost ? `Re-import and remove ${many(lost, 'link')}` : 'Re-import',
  same: 'This is the same file the IED already uses, so nothing would change.',
  counts: (what: string, c: Counts) => `${what}: ${c.before.toLocaleString()} → ${c.after.toLocaleString()}`
    + (c.added || c.removed ? ` (${c.added.toLocaleString()} new, ${c.removed.toLocaleString()} gone)` : ' (same)'),
  kept: (count: number) => `${many(count, 'link')} to this IED stay as they are.`,
  lostTitle: (count: number) => `${many(count, 'link')} cannot stay and will be removed:`,
  nothingLost: 'Nothing is lost: every link to this IED is still in the new file.',
  gateway: (points: string[]) => `These gateway points stay in the list but will not update, because the new file `
    + `no longer has them: ${points.slice(0, 8).join(', ')}${points.length > 8 ? ` and ${points.length - 8} more` : ''}.`,
  backup: 'A backup of the whole project is saved under data/recovery first, and Undo takes the re-import back.',
};
