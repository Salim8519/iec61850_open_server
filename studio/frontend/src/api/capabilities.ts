/** Speed lane F4: what the backend offers beyond polling (ADR-0087).
 *
 * The Go engine answers `GET /api/v1/capabilities` with its push stream and the polled paths each
 * topic replaces. The Python engine answers its JSON 404; then nothing changes: the app keeps
 * polling and the plain `/api/v1/events` stream. The request is a raw fetch, not `api()`, so that
 * 404 never shows the "restart ScadaBench" text. */
export type TopicSpec = { t: string; params: string[] };
export type Capabilities = { epoch: string; push: number; paths: Record<string, TopicSpec>; topics: string[] };
/** A polled path the push stream covers: its topic and the parameters it subscribes with. */
export type PathTopic = { t: string; params: Record<string, string>; key: string };

type Fetch = (input: string, init?: RequestInit) => Promise<{ ok: boolean; text: () => Promise<string> }>;

const isSpec = (spec: unknown): spec is TopicSpec => !!spec && typeof (spec as TopicSpec).t === 'string' &&
  Array.isArray((spec as TopicSpec).params) && (spec as TopicSpec).params.every(name => typeof name === 'string');

/** The capabilities when the backend offers push protocol 1; null otherwise (Python, an error, an
 * unknown protocol): the caller then keeps today's polling. Never throws. */
export async function fetchCapabilities(fetcher: Fetch, origin: string): Promise<Capabilities | null> {
  try {
    const response = await fetcher('/api/v1/capabilities', {
      credentials: 'same-origin', headers: { 'X-Studio-Frontend': origin } });
    if (!response.ok) return null;
    return readCapabilities(await response.text());
  } catch {
    return null;
  }
}

/** Parse a capabilities answer; null unless it announces push protocol 1 with a usable path map. */
export function readCapabilities(text: string): Capabilities | null {
  let data: unknown;
  try { data = JSON.parse(text); } catch { return null; }
  const caps = data as Partial<Capabilities> | null;
  if (!caps || caps.push !== 1 || typeof caps.epoch !== 'string' || !caps.paths || typeof caps.paths !== 'object') {
    return null;
  }
  const paths: Record<string, TopicSpec> = {};
  for (const [path, spec] of Object.entries(caps.paths)) if (isSpec(spec)) paths[path] = spec;
  return { epoch: caps.epoch, push: 1, paths, topics: Array.isArray(caps.topics) ? caps.topics : [] };
}

/** The topic that replaces polling `path` (relative to /api/v1, query included), or null.
 * Covered: the path without its query is in `paths` and the query names only that topic's
 * parameters, each once. `/tools/gateway/points?protocol=iec104` is covered; a traffic page
 * `?after=N` is not, so it stays polled. */
export function pathTopic(paths: Record<string, TopicSpec>, path: string): PathTopic | null {
  const mark = path.indexOf('?');
  const base = mark < 0 ? path : path.slice(0, mark);
  if (!Object.prototype.hasOwnProperty.call(paths, base)) return null;
  const spec = paths[base];
  const params: Record<string, string> = {};
  if (mark >= 0) {
    for (const [name, value] of new URLSearchParams(path.slice(mark + 1))) {
      if (!spec.params.includes(name) || Object.prototype.hasOwnProperty.call(params, name)) return null;
      params[name] = value;
    }
  }
  const key = JSON.stringify(Object.keys(params).sort().map(name => [name, params[name]]));
  return { t: spec.t, params, key };
}
