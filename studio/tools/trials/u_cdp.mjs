/* Screen trials U1-U5: a headless Chromium driven over the DevTools protocol (no extra packages).
 *
 *   const browser = await launch({ port: 22901 });   // Chrome, 1366x768, own profile
 *   const page = await browser.page(url);            // one tab, Network + Performance on
 *   await page.eval('document.title');
 *   page.requests()                                  // [{ url, method, at }] since the last reset
 *
 * Trial-only (not in verify.py). Node 24 (global WebSocket). */
import { spawn } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const CHROME = process.env.TRIAL_CHROME || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const wait = ms => new Promise(resolve => setTimeout(resolve, ms));

export async function launch({ port = 22901, width = 1366, height = 768, profileRoot = tmpdir() } = {}) {
  // The previous round's Chrome may still hold the port for a moment after it was told to quit.
  for (let tries = 0; tries < 100; tries += 1) {
    if (!await fetch(`http://127.0.0.1:${port}/json/version`).then(() => true, () => false)) break;
    await wait(100);
  }
  const profile = mkdtempSync(join(profileRoot, 'trial-chrome-'));
  const child = spawn(CHROME, ['--headless=new', `--remote-debugging-port=${port}`, `--user-data-dir=${profile}`,
    `--window-size=${width},${height}`, '--no-first-run', '--no-default-browser-check', '--disable-extensions',
    '--disable-background-timer-throttling', '--disable-renderer-backgrounding', 'about:blank'], { stdio: 'ignore' });
  let version = null;
  for (let tries = 0; tries < 300 && !version; tries += 1) {
    version = await fetch(`http://127.0.0.1:${port}/json/version`).then(r => r.json()).catch(() => null);
    if (!version) await wait(100);
  }
  if (!version) throw new Error('Chrome did not open its DevTools port');
  async function page(url) {
    const target = await fetch(`http://127.0.0.1:${port}/json/new?about:blank`, { method: 'PUT' }).then(r => r.json());
    const tab = await connect(target.webSocketDebuggerUrl);
    await tab.send('Network.enable');
    await tab.send('Performance.enable');
    await tab.send('Page.enable');
    await tab.send('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: false });
    if (url) await tab.go(url);
    return tab;
  }
  function close() {
    child.kill();
    setTimeout(() => { try { rmSync(profile, { recursive: true, force: true }); } catch { /* still locked */ } }, 1500);
  }
  return { page, close };
}

async function connect(wsUrl) {
  const socket = new WebSocket(wsUrl);
  await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject; });
  let id = 0;
  const calls = new Map();
  const listeners = new Map();
  let log = [];
  const open = new Map();
  socket.onmessage = message => {
    const data = JSON.parse(message.data);
    if (data.id && calls.has(data.id)) {
      const { resolve, reject } = calls.get(data.id);
      calls.delete(data.id);
      if (data.error) reject(new Error(data.error.message)); else resolve(data.result);
      return;
    }
    if (data.method === 'Network.requestWillBeSent' && !data.params.request.url.startsWith('data:')) {
      const url = new URL(data.params.request.url);
      log.push({ path: url.pathname + url.search, method: data.params.request.method, at: performance.now(),
        type: data.params.type });
      // The push/event stream never finishes: it does not count as an open request.
      if (data.params.type !== 'EventSource') open.set(data.params.requestId, performance.now());
    }
    if (data.method === 'Network.loadingFinished' || data.method === 'Network.loadingFailed') open.delete(data.params.requestId);
    (listeners.get(data.method) ?? []).forEach(listener => listener(data.params));
  };
  function send(method, params = {}) {
    id += 1;
    socket.send(JSON.stringify({ id, method, params }));
    return new Promise((resolve, reject) => calls.set(id, { resolve, reject }));
  }
  function once(method, timeout = 30000) {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error(`timeout waiting for ${method}`)), timeout);
      const list = listeners.get(method) ?? [];
      const listener = params => { clearTimeout(timer); listeners.set(method, (listeners.get(method) ?? []).filter(l => l !== listener)); resolve(params); };
      listeners.set(method, [...list, listener]);
    });
  }
  async function evaluate(expression) {
    const result = await send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true });
    if (result.exceptionDetails) throw new Error(result.exceptionDetails.exception?.description ?? result.exceptionDetails.text);
    return result.result.value;
  }
  async function go(url) {
    const loaded = once('Page.loadEventFired');
    await send('Page.navigate', { url });
    await loaded;
  }
  /** Wait until `expression` is truthy (checked every 20 ms). */
  async function until(expression, timeout = 30000) {
    const start = Date.now();
    while (Date.now() - start < timeout) {
      if (await evaluate(expression).catch(() => false)) return;
      await wait(20);
    }
    throw new Error(`timeout: ${expression}`);
  }
  /** Wait until no request is open for `quiet` ms. */
  async function settle(quiet = 400, timeout = 30000) {
    const start = Date.now();
    let calm = Date.now();
    while (Date.now() - start < timeout) {
      if (open.size) calm = Date.now();
      if (Date.now() - calm >= quiet) return;
      await wait(20);
    }
  }
  async function metrics() {
    const { metrics: list } = await send('Performance.getMetrics');
    return Object.fromEntries(list.map(item => [item.name, item.value]));
  }
  async function screenshot(file) {
    const { data } = await send('Page.captureScreenshot', { format: 'png' });
    writeFileSync(file, Buffer.from(data, 'base64'));
  }
  /** Call `listener` for every `method` event (for example Network.eventSourceMessageReceived). */
  function on(method, listener) { listeners.set(method, [...(listeners.get(method) ?? []), listener]); }
  return { send, once, on, eval: evaluate, go, until, settle, metrics, screenshot,
    requests: () => log, reset: () => { log = []; }, openCount: () => open.size, close: () => socket.close() };
}

export const median = values => {
  const s = [...values].sort((a, b) => a - b);
  const mid = Math.floor(s.length / 2);
  return !s.length ? null : s.length % 2 ? s[mid] : (s[mid - 1] + s[mid]) / 2;
};
export const worst = values => values.length ? Math.max(...values) : null;
