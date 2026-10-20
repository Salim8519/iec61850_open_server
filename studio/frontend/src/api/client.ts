import { rememberAppWindow } from '../app/useExitConfirm';

async function send(path: string, method = 'GET', body?: unknown) {
  const form = body instanceof FormData;
  const response = await fetch(`/api/v1${path}`, {
    method, credentials: 'same-origin',
    headers: { 'X-Studio-Frontend': window.location.origin,
      ...(body && !form ? { 'Content-Type': 'application/json' } : {}) },
    body: body ? (form ? body : JSON.stringify(body)) : undefined,
  });
  return { response, text: await response.text() };
}

/** The answer's JSON, or the same errors and session/licence handling for every caller. */
function answer(response: Response, text: string): unknown {
  const data = JSON.parse(text);
  if (response.status === 403 && typeof data.active_frontend === 'string') {
    const active = new URL(data.active_frontend);
    if (active.hostname === '127.0.0.1' || active.hostname === 'localhost') {
      window.location.replace(`${active.origin}${window.location.pathname}${window.location.hash}`);
      throw new Error('Switching this tab to the active ScadaBench session.');
    }
  }
  if (response.status === 402) {
    window.dispatchEvent(new Event('studio-license-required'));
    throw new Error(data.error || 'A license is required.');
  }
  if (response.status === 404) throw new Error('The running backend does not provide this action. Save your project and restart ScadaBench to load the updated backend.');
  if (!response.ok) throw Object.assign(new Error(data.error || `Request failed (${response.status})`), { data });
  return data;
}

export async function api<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
  const { response, text } = await send(path, method, body);
  return answer(response, text) as T;
}

/** GET as raw text, so the live feed can skip unchanged answers without parsing them. */
export async function apiText(path: string): Promise<string> {
  const { response, text } = await send(path);
  if (!response.ok) answer(response, text);
  return text;
}

export async function connectSession(): Promise<void> {
  const fragment = new URLSearchParams(window.location.hash.slice(1));
  const token = fragment.get('session');
  rememberAppWindow(fragment);
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
