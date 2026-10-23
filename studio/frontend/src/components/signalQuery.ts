export type SignalQuery = {
  q?: string; reportRef?: string; offset?: number; limit?: number;
  kinds?: string[]; fc?: string; writableOnly?: boolean;
  /** Universal point type (SP, DP, MV …) the engine filters by; "all" or empty sends nothing (ADR-0095). */
  pointKind?: string;
  /** Go only (ADR-0113): hide the q/t helpers, count per type and per object, give each row its type. */
  smart?: boolean;
  /** Show the quality and time helpers (smart pickers hide them by default). */
  helpers?: boolean;
  /** One object (logical node "LD/LN"); empty means every object. */
  ln?: string;
  /** The component's name: the engine suggests the points that mention it. */
  near?: string;
};

export const SIGNAL_PAGE = 50;

/** Shared query builder for IED-scoped signal search lists. */
export function buildSignalQuery(query: SignalQuery): string {
  const params = new URLSearchParams({
    q: query.q || '',
    offset: String(query.offset || 0),
    limit: String(query.limit || SIGNAL_PAGE),
  });
  if (query.reportRef && query.reportRef !== '__all__') params.set('report', query.reportRef);
  if (query.kinds?.length) params.set('kind', query.kinds.join(','));
  if (query.fc) params.set('fc', query.fc);
  if (query.writableOnly) params.set('writable', 'true');
  if (query.pointKind && query.pointKind !== 'all') params.set('point_kind', query.pointKind);
  if (query.smart) {
    if (!query.helpers) params.set('helpers', 'hide');
    params.set('kind_counts', '1');
    params.set('ln_counts', '1');
    params.set('annotate', '1');
    if (query.ln) params.set('ln', query.ln);
    if (query.near?.trim()) params.set('near', query.near.trim());
  }
  return params.toString();
}

/** Storage key for an IED signal across pickers and live values. */
export function signalKey(iedId: string, ref: string): string {
  return `${iedId}:${ref}`;
}
