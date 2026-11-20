/* Screen trials U1-U5 (2026-11-20): a tiny HTTP client for one engine instance.
 *
 * `session(url)` signs in with the launch token printed by studio_server.exe ("Open in your
 * browser: http://127.0.0.1:<port>#session=<token>") and returns `call(path, method, body)`.
 * Trial-only (not in verify.py). Node 24. */
import { readFile } from 'node:fs/promises';

export async function session(launch) {
  const [origin, fragment] = launch.split('#');
  const token = new URLSearchParams(fragment).get('session');
  let cookie = '';
  async function call(path, method = 'GET', body) {
    const response = await fetch(`${origin}/api/v1${path}`, {
      method, headers: { 'X-Studio-Frontend': origin, ...(cookie ? { Cookie: cookie } : {}),
        ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}) },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
    const set = response.headers.get('set-cookie');
    if (set) cookie = set.split(';')[0];
    const text = await response.text();
    if (!response.ok) throw new Error(`${method} ${path}: ${response.status} ${text.slice(0, 300)}`);
    return text ? JSON.parse(text) : null;
  }
  await call('/session', 'POST', { token });
  return { call, origin, token };
}

/** The launch address an engine printed into its log file. */
export async function launchUrl(logFile) {
  for (let tries = 0; tries < 100; tries += 1) {
    const text = await readFile(logFile, 'utf8').catch(() => '');
    const found = /Open in your browser: (\S+)/.exec(text);
    if (found) return found[1];
    await new Promise(resolve => setTimeout(resolve, 200));
  }
  throw new Error(`no launch address in ${logFile}`);
}
