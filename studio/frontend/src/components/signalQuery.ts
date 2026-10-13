export type SignalQuery = {
  q?: string; reportRef?: string; offset?: number; limit?: number;
  kinds?: string[]; fc?: string; writableOnly?: boolean;
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
  return params.toString();
}

/** Storage key for an IED signal across pickers and live values. */
export function signalKey(iedId: string, ref: string): string {
  return `${iedId}:${ref}`;
}
