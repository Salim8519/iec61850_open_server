/** Speed lane F4: topic frames of the Go push stream turned back into the exact GET text.
 *
 * Pure (no DOM, no timers): the push client hands every `event: push` frame to `apply` and shows
 * the text it gets back. The schema is in docs/architecture/go-engine/streams/speed-g.md; the Go
 * reference of this algorithm is studio/engine/internal/push/mirror_test.go.
 *
 * - A full frame's body is the exact text after `"full":` up to the frame's last "}". It is cut
 *   out, never re-encoded (JSON.stringify would lose `1.0`, escapes and key order); only the
 *   members before it are parsed. A full frame is always applied.
 * - A delta (keyed lists) replaces head/tail, removes keys, appends added keys in order, upserts
 *   row texts and rebuilds `head + rows.join(",") + tail`. It is applied when its base is the
 *   held revision, ignored when its revision is not newer, otherwise the topic asks for resync
 *   and ignores deltas until the next full frame. */
export type Outcome =
  | { kind: 'hello'; epoch: string; client: string }
  | { kind: 'text'; t: string; text: string }
  | { kind: 'status'; t: string; status: number }
  | { kind: 'skip'; t: string }
  | { kind: 'resync'; t: string }
  | { kind: 'bad' };

type Rows = { head: number; keys: string[]; ends: number[] };
type FullMeta = { t: string; rev: number; status?: number; rows?: Rows };
type Delta = { t: string; rev: number; base: number; head?: string; tail?: string;
  upsert?: Record<string, string>; remove?: string[]; add?: string[] };
type State = { rev: number; text: string | null; keyed: boolean; head: string; tail: string; keys: string[];
  rows: Map<string, string>; awaiting: boolean; gate: number; dropped: boolean };

const HELLO = '{"t":"hello",';
// Member order is fixed: t, rev, then base for a delta; t, rev, [status], [rows], full for a full frame.
const DELTA = /^\{"t":"[^"\\]*","rev":-?\d+,"base":/;

const fresh = (gate = 0): State => ({ rev: 0, text: null, keyed: false, head: '', tail: '', keys: [],
  rows: new Map(), awaiting: true, gate, dropped: false });

/** The exact body text of a full frame, or null when the frame has none. */
export function fullText(frame: string): string | null {
  const at = frame.indexOf('"full":');
  return at < 0 ? null : frame.slice(at + 7, -1);
}

/** The server counts offsets in bytes (UTF-8). Flask bodies are ASCII, where bytes are string
 * indexes; a body with other characters is converted, so a cut never lands inside a row. */
function toIndexes(text: string, offsets: number[]): number[] {
  if (!/[^\x00-\x7f]/.test(text)) return offsets;
  let bytes = 0;
  let index = 0;
  return offsets.map(offset => {
    while (bytes < offset && index < text.length) {
      const code = text.codePointAt(index) as number;
      bytes += code < 0x80 ? 1 : code < 0x800 ? 2 : code < 0x10000 ? 3 : 4;
      index += code >= 0x10000 ? 2 : 1;
    }
    return index;
  });
}

function applyFull(state: State, meta: FullMeta, body: string) {
  state.rev = meta.rev;
  state.text = body;
  state.awaiting = false;
  const rows = meta.rows;
  state.keyed = !!rows && Array.isArray(rows.keys) && Array.isArray(rows.ends) && rows.keys.length === rows.ends.length;
  if (!rows || !state.keyed) return;
  const [head, ...ends] = toIndexes(body, [rows.head, ...rows.ends]);
  state.head = body.slice(0, head);
  state.keys = rows.keys.slice();
  state.rows = new Map();
  let start = head;
  ends.forEach((end, index) => {
    state.rows.set(state.keys[index], body.slice(start, end));
    start = end + 1;
  });
  if (rows.ends.length) start -= 1;
  state.tail = body.slice(start);
}

/** Apply a delta whose base is the held revision; false when a row text is missing (then resync). */
function applyDelta(state: State, delta: Delta): boolean {
  if (delta.head !== undefined) state.head = delta.head;
  if (delta.tail !== undefined) state.tail = delta.tail;
  const removed = new Set(delta.remove ?? []);
  removed.forEach(key => state.rows.delete(key));
  const kept = removed.size ? state.keys.filter(key => !removed.has(key)) : state.keys;
  state.keys = delta.add?.length ? kept.concat(delta.add) : kept;
  for (const [key, row] of Object.entries(delta.upsert ?? {})) state.rows.set(key, row);
  const parts = new Array<string>(state.keys.length);
  for (let index = 0; index < state.keys.length; index += 1) {
    const row = state.rows.get(state.keys[index]);
    if (row === undefined) return false;
    parts[index] = row;
  }
  state.rev = delta.rev;
  state.text = state.head + parts.join(',') + state.tail;
  return true;
}

/** The held text of every tracked topic of one tab. */
export function createTopicStates() {
  const states = new Map<string, State>();

  /** Start (or restart, after a parameter change) holding `t`; frames wait for its full frame.
   * With `gate` > 0, frames are dropped until PUT number `gate` is answered: until then the
   * stream may still carry frames of the old parameters. */
  function track(t: string, gate = 0) { states.set(t, fresh(gate)); }

  function untrack(t: string) { states.delete(t); }

  /** The server restarted (new epoch): every topic waits for its full frame again. */
  function resetAll() { states.forEach((state, t) => states.set(t, fresh(state.gate))); }

  /** PUT number `number` was answered: open its gates. Returns the topics that dropped a full
   * frame while gated and still hold none (they ask for resync). */
  function acked(number: number): string[] {
    const ask: string[] = [];
    states.forEach((state, t) => {
      if (!state.gate || state.gate > number) return;
      if (state.dropped && state.awaiting) ask.push(t);
      state.gate = 0;
      state.dropped = false;
    });
    return ask;
  }

  function apply(frame: string): Outcome {
    try {
      if (frame.startsWith(HELLO)) {
        const hello = JSON.parse(frame) as { epoch: string; client: string };
        return { kind: 'hello', epoch: String(hello.epoch), client: String(hello.client) };
      }
      const at = DELTA.test(frame) ? -1 : frame.indexOf('"full":');
      if (at > 0) return full(JSON.parse(`${frame.slice(0, at - 1)}}`) as FullMeta, frame.slice(at + 7, -1));
      const delta = JSON.parse(frame) as Delta;
      return typeof delta.base === 'number' ? applyOne(delta) : { kind: 'bad' };
    } catch {
      return { kind: 'bad' };
    }
  }

  function full(meta: FullMeta, body: string): Outcome {
    const state = states.get(meta.t);
    if (!state) return { kind: 'skip', t: meta.t };
    if (state.gate) { state.dropped = true; return { kind: 'skip', t: meta.t }; }
    applyFull(state, meta, body);
    if (meta.status !== undefined && meta.status !== 200) {
      state.keyed = false;
      return { kind: 'status', t: meta.t, status: meta.status };
    }
    return { kind: 'text', t: meta.t, text: body };
  }

  function applyOne(delta: Delta): Outcome {
    const state = states.get(delta.t);
    if (!state || state.gate || state.awaiting) return { kind: 'skip', t: delta.t };
    if (delta.base === state.rev && state.keyed) {
      if (applyDelta(state, delta)) return { kind: 'text', t: delta.t, text: state.text as string };
    } else if (delta.rev <= state.rev) {
      return { kind: 'skip', t: delta.t };
    }
    state.awaiting = true;
    return { kind: 'resync', t: delta.t };
  }

  return { track, untrack, resetAll, acked, apply,
    tracked: (t: string) => states.has(t),
    text: (t: string) => states.get(t)?.text ?? null,
    rev: (t: string) => states.get(t)?.rev ?? 0 };
}

export type TopicStates = ReturnType<typeof createTopicStates>;
