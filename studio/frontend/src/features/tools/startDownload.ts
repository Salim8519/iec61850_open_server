/** Starts the browser's own download of a file the engine serves with a plain GET (a CSV export). */
export function startDownload(url: string) {
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = '';
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
}
