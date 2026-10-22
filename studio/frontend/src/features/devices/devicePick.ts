/** Small choices of the Devices list (change 0177), kept apart from the page so they are easy to test. */

/** The first "SampleN" no IED uses yet (names compared without case), so "Add a sample IED" never asks
 *  for a name the project already has, for example after an IED was deleted. */
export function freeSampleName(ieds: { name: string }[], stem = 'Sample'): string {
  const used = new Set(ieds.map(ied => ied.name.toLowerCase()));
  for (let number = 1; ; number++) {
    const name = `${stem}${number}`;
    if (!used.has(name.toLowerCase())) return name;
  }
}

/** The IED the detail pane should show: the chosen one while the list shows it (or it was asked for from
 *  elsewhere), else the first IED the filter shows, so the detail never shows an IED the list hides. */
export function shownSelection(selected: string, shownIds: string[], requested?: string): string {
  if (!shownIds.length || shownIds.includes(selected) || selected === requested) return selected;
  return shownIds[0];
}
