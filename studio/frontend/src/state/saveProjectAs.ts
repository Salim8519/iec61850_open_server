/** "Save as…": the open project written to a file the user names, in a folder they pick (the
 *  Windows Save dialog). Where the browser has no such dialog, it downloads under that name. */

const DOWNLOAD = '/api/v1/project/download';
const EXTENSION = '.scadabench';

type SaveHandle = { createWritable: () => Promise<{ write: (data: Blob) => Promise<void>; close: () => Promise<void> }> };
type SavePicker = (options: { suggestedName: string; types: { description: string; accept: Record<string, string[]> }[] })
  => Promise<SaveHandle>;

/** A file name from the project name: characters Windows refuses become spaces. */
export function projectFileName(name: string): string {
  const base = name.replace(/[<>:"/\\|?*\u0000-\u001f]/g, ' ').replace(/\s+/g, ' ').trim().replace(/[. ]+$/, '');
  return `${base || 'substation'}${EXTENSION}`;
}

/** Saves a copy; answers whether it was saved, downloaded or cancelled by the user. */
export async function saveProjectAs(name: string): Promise<'saved' | 'downloaded' | 'cancelled'> {
  const suggestedName = projectFileName(name);
  const picker = (window as unknown as { showSaveFilePicker?: SavePicker }).showSaveFilePicker;
  if (!picker) {
    const link = document.createElement('a');
    link.href = DOWNLOAD; link.download = suggestedName; link.click();
    return 'downloaded';
  }
  let handle: SaveHandle;
  try {
    handle = await picker({ suggestedName, types: [{ description: 'ScadaBench project', accept: { 'application/zip': [EXTENSION] } }] });
  } catch (reason) {
    if (reason instanceof DOMException && reason.name === 'AbortError') return 'cancelled';
    throw reason;
  }
  const response = await fetch(DOWNLOAD, { credentials: 'same-origin' });
  if (!response.ok) throw new Error('The project could not be read to save it. Try again.');
  const file = await handle.createWritable();
  await file.write(await response.blob());
  await file.close();
  return 'saved';
}
