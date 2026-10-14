/** Plain-English wording for the Tidy up action (beginner guidance rules). */

export const TIDY_LOCKED = 'Stop the run to rearrange the diagram.';

export function tidyTitle(locked: boolean, scoped: boolean): string {
  if (locked) return TIDY_LOCKED;
  return scoped
    ? 'Arrange only the selected items like a standard single-line diagram. Undo puts them back.'
    : 'Arrange the whole diagram like a standard single-line diagram: busbars on top, bays straight down, ' +
      'relay panels and buttons to the side. Wires and links stay the same. Undo puts it back.';
}

export function tidyMessage(changed: boolean, moved: number, scoped: boolean): string {
  if (!changed) return scoped ? 'The selection is already tidy.' : 'The diagram is already tidy.';
  if (!moved) return 'Straightened the wires. Press Undo to put them back.';
  return `Tidied ${moved} item${moved === 1 ? '' : 's'}. Press Undo to put ${moved === 1 ? 'it' : 'them'} back.`;
}
