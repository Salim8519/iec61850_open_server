/** Download an authenticated, server-verified artifact without mutating the project. */
export async function download(path: string, body: unknown, filename: string): Promise<void> {
  const response = await fetch(`/api/v1${path}`, {
    method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    if (response.status === 404) throw new Error('The running backend does not provide this export. Save your project and restart Studio to load the updated backend.');
    const result = await response.json();
    throw new Error(result.error || `Download failed (${response.status})`);
  }
  const url = URL.createObjectURL(await response.blob());
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 60000);
}
