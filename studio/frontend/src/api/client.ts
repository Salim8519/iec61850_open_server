export async function api<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
  const form = body instanceof FormData;
  const response = await fetch(`/api/v1${path}`, {
    method, credentials: 'same-origin',
    headers: { 'X-Studio-Frontend': window.location.origin,
      ...(body && !form ? { 'Content-Type': 'application/json' } : {}) },
    body: body ? (form ? body : JSON.stringify(body)) : undefined,
  });
  const data = await response.json();
  if (response.status === 403 && typeof data.active_frontend === 'string') {
    const active = new URL(data.active_frontend);
    if (active.hostname === '127.0.0.1' || active.hostname === 'localhost') {
      window.location.replace(`${active.origin}${window.location.pathname}${window.location.hash}`);
      throw new Error('Switching this tab to the active Studio session.');
    }
  }
  if (response.status === 404) throw new Error('The running backend does not provide this action. Save your project and restart Studio to load the updated backend.');
  if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
  return data as T;
}

export async function connectSession(): Promise<void> {
  const fragment = new URLSearchParams(window.location.hash.slice(1));
  const token = fragment.get('session');
  if (token) {
    await api('/session', 'POST', { token });
    window.history.replaceState(null, '', window.location.pathname);
  }
}

export async function upload(path: string, file: File) {
  const form = new FormData();
  form.append('file', file);
  return api(path, 'POST', form);
}
