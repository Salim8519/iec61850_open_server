// One debounced signal search for a picker (ADR-0113): the engine filters, counts and suggests;
// this hook only asks and keeps the latest answer.
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import { buildSignalQuery, type SignalQuery } from '../signalQuery';
import type { PickerAnswer } from './pickerAnswer';

export function useSignalSearch(iedId: string, request: SignalQuery) {
  const [answer, setAnswer] = useState<PickerAnswer>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const query = buildSignalQuery(request);
  useEffect(() => {
    if (!iedId) { setAnswer(undefined); return undefined; }
    let cancelled = false;
    const timer = setTimeout(() => {
      setLoading(true);
      api<PickerAnswer>(`/ieds/${iedId}/signals?${query}`)
        .then(data => { if (!cancelled) { setAnswer(data); setError(''); } })
        .catch(reason => { if (!cancelled) setError(reason.message); })
        .finally(() => { if (!cancelled) setLoading(false); });
    }, 150);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [iedId, query]);
  return { answer, loading, error };
}
