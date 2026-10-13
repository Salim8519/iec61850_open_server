import { useEffect, useRef, useState } from 'react';
import type { Ied, Signal, SignalBinding } from '../api/types';
import { api } from '../api/client';
import './signal-picker.css';

type Props = { ieds: Ied[]; disabled?: boolean; value?: SignalBinding;
  allowedKinds?: string[]; allowedFc?: string; writableOnly?: boolean;
  choose: (binding: SignalBinding, signal: Signal) => void };

export function SignalPicker({ ieds, disabled, value, allowedKinds, allowedFc, writableOnly = true, choose }: Props) {
  const [iedId, setIed] = useState(value?.ied_id || '');
  const [reportRef, setReport] = useState(value ? value.report_ref || '__all__' : '');
  const [offset, setOffset] = useState(0);
  const [items, setItems] = useState<Signal[]>([]);
  const [selected, setSelected] = useState<Signal | null>(null);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const selectionRevision = useRef(0);
  const ied = ieds.find(item => item.id === iedId);
  const report = ied?.reports.find(item => item.ref === reportRef);
  const kinds = allowedKinds?.join(',') || '';
  useEffect(() => {
    setIed(value?.ied_id || ''); setReport(value ? value.report_ref || '__all__' : '');
    setSelected(null); setOffset(0);
    if (!value?.ref) return;
    const revision = ++selectionRevision.current;
    let cancelled = false;
    api<{ items: Signal[] }>(`/ieds/${value.ied_id}/signals?ref=${encodeURIComponent(value.ref)}`)
      .then(data => { if (!cancelled && revision === selectionRevision.current) setSelected(data.items[0] || null); })
      .catch(reason => { if (!cancelled) setError(reason.message); });
    return () => { cancelled = true; };
  }, [value?.ied_id, value?.report_ref, value?.ref]);
  useEffect(() => {
    setItems([]); setTotal(0);
    if (!iedId || !reportRef) return;
    let cancelled = false;
    setLoading(true);
    const params = new URLSearchParams({ offset: String(offset), limit: '100' });
    if (writableOnly) params.set('writable', 'true');
    if (reportRef !== '__all__') params.set('report', reportRef);
    if (kinds) params.set('kind', kinds);
    if (allowedFc) params.set('fc', allowedFc);
    api<{ items: Signal[]; total: number }>(`/ieds/${iedId}/signals?${params}`)
      .then(data => { if (!cancelled) { setItems(data.items); setTotal(data.total); setError(''); } })
      .catch(reason => { if (!cancelled) setError(reason.message); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [iedId, reportRef, offset, kinds, allowedFc]);
  return <fieldset className="signal-picker stack" disabled={disabled}>
    <legend>IED / report / signal / exact path</legend>
    <label>IED<select value={iedId} onChange={e => {
      selectionRevision.current++; setIed(e.target.value); setReport(''); setSelected(null); setOffset(0);
    }}><option value="">Choose an IED</option>{ieds.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
    {ied && <label>Report<select value={reportRef} onChange={e => {
      selectionRevision.current++; setReport(e.target.value); setSelected(null); setOffset(0);
    }}><option value="">Choose a report first</option>
      <option value="__all__">All signals / not report-filtered</option>
      {ied.reports.map(item => <option key={item.ref} value={item.ref}>
        {item.ref} | {item.buffered ? 'Buffered' : 'Unbuffered'} | {item.signal_count} signals
      </option>)}
    </select></label>}
    {report && <div className="small muted">Dataset: <code className="reference-block">{report.dataset_ref || 'Not assigned'}</code>
      {!!report.unresolved.length && <span role="status">{report.unresolved.length} unresolved dataset members are not selectable.</span>}
    </div>}
    {ied && reportRef && <>
      <label>Signal<select value={selected?.ref || ''} onChange={e => {
        const signal = items.find(item => item.ref === e.target.value) || null;
        setSelected(signal);
        if (signal) choose({ ied_id: iedId,
          report_ref: reportRef === '__all__' ? null : reportRef, ref: signal.ref }, signal);
      }}>
        <option value="">{loading ? 'Loading signals...' : 'Choose a signal'}</option>
        {selected && !items.some(item => item.ref === selected.ref) && <option value={selected.ref}>{selected.description || selected.ref}</option>}
        {items.map(item => <option key={item.ref} value={item.ref}>{item.description || item.ref} | {item.ref} [{item.fc}]</option>)}
      </select></label>
      <div className="row small"><button type="button" disabled={loading || offset === 0} onClick={() => setOffset(offset - 100)}>Previous</button>
        <span>{total ? offset + 1 : 0}-{Math.min(offset + 100, total)} / {total}</span>
        <button type="button" disabled={loading || offset + 100 >= total} onClick={() => setOffset(offset + 100)}>Next</button></div>
      {selected && <><code className="reference-block">{selected.ref}</code>
        <div className="small">{selected.kind} | {selected.fc} | Assigned when selected</div></>}
    </>}
    {error && <p role="alert">{error}</p>}
  </fieldset>;
}
