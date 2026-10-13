import { useEffect, useState } from 'react';
import { Search } from 'lucide-react';
import type { Ied, Signal, SignalBinding } from '../api/types';
import { api } from '../api/client';
import { buildSignalQuery, signalKey } from './signalQuery';
import { shortRef } from '../equipment/relayDisplay';
import './signal-search-picker.css';

type Props = {
  ieds: Ied[]; fixedIedId?: string; allowedKinds?: string[]; allowedFc?: string;
  writableOnly?: boolean; exclude?: Set<string>; disabled?: boolean; chooseLabel?: string;
  choose: (binding: SignalBinding, signal: Signal) => void;
};

/** Search-first signal chooser: pick an IED once, then search and one-click add. */
export function SignalSearchPicker({ ieds, fixedIedId, allowedKinds, allowedFc,
  writableOnly, exclude, disabled, chooseLabel = 'Choose', choose }: Props) {
  const onlyIed = ieds.length === 1 ? ieds[0].id : '';
  const [iedId, setIed] = useState(fixedIedId || onlyIed);
  const [query, setQuery] = useState('');
  const [reportRef, setReport] = useState('__all__');
  const [offset, setOffset] = useState(0);
  const [items, setItems] = useState<Signal[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const ied = ieds.find(item => item.id === iedId);
  useEffect(() => { setIed(fixedIedId || onlyIed); }, [fixedIedId, onlyIed]);
  useEffect(() => { setOffset(0); }, [query, reportRef, iedId, fixedIedId]);
  useEffect(() => {
    if (!iedId) { setItems([]); setTotal(0); return; }
    let cancelled = false;
    const timer = setTimeout(() => {
      setLoading(true);
      api<{ items: Signal[]; total: number }>(
        `/ieds/${iedId}/signals?${buildSignalQuery({ q: query, reportRef, offset,
          kinds: allowedKinds, fc: allowedFc, writableOnly })}`)
        .then(data => { if (!cancelled) { setItems(data.items); setTotal(data.total); setError(''); } })
        .catch(reason => { if (!cancelled) setError(reason.message); })
        .finally(() => { if (!cancelled) setLoading(false); });
    }, 150);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [iedId, query, reportRef, offset, allowedKinds?.join(','), allowedFc, writableOnly]);
  return <div className="stack" style={{ gap: 10 }}>
    {!fixedIedId && <label>IED<select value={iedId} disabled={disabled}
      onChange={e => setIed(e.target.value)}>
      <option value="">Choose an IED</option>
      {ieds.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}
    </select></label>}
    {ied && <div className="row">
      <div className="search-field" style={{ flex: 1 }}><Search size={14} />
        <input aria-label="Search signals" placeholder="Search signals…" value={query}
          disabled={disabled} onChange={e => setQuery(e.target.value)} /></div>
      <select aria-label="Report" style={{ width: 140 }} value={reportRef} disabled={disabled}
        onChange={e => setReport(e.target.value)}>
        <option value="__all__">All reports</option>
        {ied.reports.map(report => <option key={report.ref} value={report.ref}>{report.ref}</option>)}
      </select>
    </div>}
    {error && <p role="alert">{error}</p>}
    {ied && <><div className="signal-search-list">
      {items.map(signal => {
        const taken = exclude?.has(signalKey(iedId, signal.ref));
        const triggerable = signal.writable && signal.fc !== 'CF';
        return <div className="signal-search-row" key={signal.ref}>
          <span className={`signal-dot ${triggerable ? 'writable' : 'readonly'}`} aria-hidden="true" />
          <span className="signal-search-text" title={signal.ref}>
            <strong>{signal.description || shortRef(signal.ref, ied?.name)}</strong>
            <small>{signal.description ? `${shortRef(signal.ref, ied?.name)} · ` : ''}{signal.kind} · {signal.fc}</small></span>
          {taken ? <span className="badge live">Added ✓</span> :
            <button className={triggerable ? 'primary' : ''} disabled={disabled || loading}
              title={triggerable ? 'Choose this signal' : 'Choose as display-only'}
              onClick={() => choose({ ied_id: iedId,
                report_ref: reportRef === '__all__' ? null : reportRef, ref: signal.ref }, signal)}>
              {chooseLabel}</button>}
        </div>;
      })}
      {!loading && !items.length && <p className="small muted">No signals match.</p>}
    </div>
    <div className="row small muted"><span>{total ? offset + 1 : 0}–{Math.min(offset + 50, total)} of {total}</span>
      <span style={{ flex: 1 }} />
      <button type="button" className="ghost" disabled={loading || offset === 0}
        onClick={() => setOffset(offset - 50)}>Previous</button>
      <button type="button" className="ghost" disabled={loading || offset + 50 >= total}
        onClick={() => setOffset(offset + 50)}>Next</button></div></>}
  </div>;
}
