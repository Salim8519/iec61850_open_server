/** Starts a download without leaving the page (a link click, so the app window's exit prompt does not appear). */
export function saveFile(url: string) {
  const link = document.createElement('a');
  link.href = url;
  link.download = '';
  link.click();
}
