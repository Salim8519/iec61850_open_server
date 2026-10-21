export type SignalQuery = {
  q?: string; reportRef?: string; offset?: number; limit?: number;
  kinds?: string[]; fc?: string; writableOnly?: boolean;
  /** Universal point type (SP, DP, MV …) the engine filters by; "all" or empty sends nothing (ADR-0095). */
  pointKind?: string;
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
  return params.toString();
}

/** Storage key for an IED signal across pickers and live values. */
export function signalKey(iedId: string, ref: string): string {
  return `${iedId}:${ref}`;
}
