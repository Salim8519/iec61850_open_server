/** Fakes for the speed lane F4 push client tests: an EventSource the test drives, a recorded
 * PUT endpoint, hand-run "soon"/"later" queues and a frame builder that follows the schema of
 * docs/architecture/go-engine/streams/speed-g.md (a JS stand-in for the Go hub). */
import type { EventSourceLike } from '../../frontend/src/api/pushClient.ts';

export class FakeSource implements EventSourceLike {
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((message: { data: string; lastEventId: string }) => void) | null = null;
  readyState = 0;
  closed = false;
  url: string;
  private push: ((message: { data: string }) => void)[] = [];
  constructor(url: string) { this.url = url; }
  addEventListener(_type: 'push', listener: (message: { data: string }) => void) { this.push.push(listener); }
  close() { this.closed = true; this.readyState = 2; }
  open() { this.readyState = 1; this.onopen?.(); }
  bus(id: number, event: object) { this.onmessage?.({ data: JSON.stringify(event), lastEventId: String(id) }); }
  frame(data: string) { this.push.forEach(listener => listener({ data })); }
  hello(epoch: string, client = 'tab-1') { this.frame(JSON.stringify({ t: 'hello', epoch, client })); }
  fail(closed: boolean) { this.readyState = closed ? 2 : 0; this.onerror?.(); }
}

/** Callbacks queued by the client; the test runs them when it decides time passed. */
export function queue() {
  const waiting: (() => void)[] = [];
  return { add: (run: () => void) => { waiting.push(run); }, run: () => waiting.splice(0).forEach(run => run()),
    size: () => waiting.length };
}

export type PutBody = { client: string; topics: { t: string; params: Record<string, string> }[]; resync: string[] };

/** PUT /api/v1/push/topics: every body is kept; each answer waits until the test releases it. */
export function topicsEndpoint() {
  const bodies: PutBody[] = [];
  const pending: { resolve: (answer: unknown) => void; reject: (error: Error) => void }[] = [];
  return {
    bodies,
    put: (body: PutBody) => { bodies.push(JSON.parse(JSON.stringify(body)) as PutBody);
      return new Promise<unknown>((resolve, reject) => { pending.push({ resolve, reject }); }); },
    answer: (unknown: string[] = []) => pending.shift()?.resolve({ epoch: 'e1', topics: [], unknown }),
    fail: () => pending.shift()?.reject(new Error('Request failed (500)')),
    inFlight: () => pending.length,
  };
}

/** A body with one keyed list, held as text pieces exactly like the Go hub cuts it. */
export type Keyed = { head: string; keys: string[]; rows: Map<string, string>; tail: string };
export const keyedText = (body: Keyed) => body.head + body.keys.map(key => body.rows.get(key)).join(',') + body.tail;

export function fullFrame(t: string, rev: number, body: Keyed | string, status?: number): string {
  const text = typeof body === 'string' ? body : keyedText(body);
  const meta: string[] = [`"t":${JSON.stringify(t)}`, `"rev":${rev}`];
  if (status !== undefined) meta.push(`"status":${status}`);
  if (typeof body !== 'string') {
    let end = body.head.length - 1;
    const ends = body.keys.map(key => (end += 1 + (body.rows.get(key) as string).length));
    meta.push(`"rows":{"head":${body.head.length},"keys":${JSON.stringify(body.keys)},"ends":${JSON.stringify(ends)}}`);
  }
  return `{${meta.join(',')},"full":${text}}`;
}

/** The delta from `before` to `after`, or null when only a full frame can say it (a reorder). */
export function deltaFrame(t: string, rev: number, base: number, before: Keyed, after: Keyed): string | null {
  const removed = before.keys.filter(key => !after.rows.has(key));
  const kept = before.keys.filter(key => after.rows.has(key));
  if (kept.some((key, index) => after.keys[index] !== key)) return null;
  const added = after.keys.slice(kept.length);
  if (added.some(key => before.rows.has(key))) return null;
  let frame = `{"t":${JSON.stringify(t)},"rev":${rev},"base":${base}`;
  if (before.head !== after.head) frame += `,"head":${JSON.stringify(after.head)}`;
  if (before.tail !== after.tail) frame += `,"tail":${JSON.stringify(after.tail)}`;
  const upsert = after.keys.filter(key => before.rows.get(key) !== after.rows.get(key));
  if (upsert.length) frame += `,"upsert":{${upsert.map(key => `${JSON.stringify(key)}:${JSON.stringify(after.rows.get(key))}`).join(',')}}`;
  if (removed.length) frame += `,"remove":${JSON.stringify(removed)}`;
  if (added.length) frame += `,"add":${JSON.stringify(added)}`;
  return `${frame}}`;
}
