import { useEffect, useState } from 'react';
import { Search } from 'lucide-react';
import type { Ied, Signal, SignalBinding } from '../api/types';
import { api } from '../api/client';
import { Spinner } from '../ui/Spinner';
import { initialView, rememberView, type PickerView } from './pickerMemory';
import { buildSignalQuery, signalKey } from './signalQuery';
import { shortRef } from '../equipment/relayDisplay';
import { useDataClasses } from '../points/dataClasses';
import { KindBadge, kindRow } from '../ui/KindBadge';
import { ALL_KINDS } from '../ui/kindFilter';
import { KindSelect } from './KindSelect';
import { kindOfSignal } from './signalKind';
import './signal-search-picker.css';

type Props = {
  ieds: Ied[]; fixedIedId?: string; allowedKinds?: string[]; allowedFc?: string;
  writableOnly?: boolean; exclude?: Set<string>; disabled?: boolean; chooseLabel?: string;
  /** Remembers IED, search, report and page under this name, so reopening continues where the user was. */
  memoryKey?: string;
  /** The point chosen now ("iedId:ref"): its row says Chosen, the others offer to replace it. */
  chosen?: SignalBinding;
  choose: (binding: SignalBinding, signal: Signal) => void;
  /** Adds every not-yet-added signal of the page in one click (measurement lists, ADR-0090). */
  chooseAll?: (items: { binding: SignalBinding; signal: Signal }[]) => void;
};

/** Search-first signal chooser: pick an IED once, then search and one-click add. */
export function SignalSearchPicker({ ieds, fixedIedId, allowedKinds, allowedFc,
  writableOnly, exclude, disabled, chooseLabel = 'Choose', memoryKey, chosen, choose, chooseAll }: Props) {
  const onlyIed = ieds.length === 1 ? ieds[0].id : '';
  const [view, setView] = useState<PickerView>(() =>
    initialView(memoryKey, fixedIedId || chosen?.ied_id, onlyIed));
  const iedId = fixedIedId || view.iedId || onlyIed;
  const { query, reportRef, offset } = view;
  const pointKind = view.pointKind ?? ALL_KINDS;
  const [items, setItems] = useState<Signal[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const ied = ieds.find(item => item.id === iedId);
  const classOf = useDataClasses(iedId);
  // A new search, report or IED starts again at the first page; paging keeps the rest.
  const change = (next: Partial<PickerView>) => setView(current => ({ ...current, offset: 0, ...next }));
  useEffect(() => { rememberView(memoryKey, { ...view, iedId }); }, [memoryKey, view, iedId]);
  useEffect(() => {
    if (!iedId) { setItems([]); setTotal(0); return; }
    let cancelled = false;
    const timer = setTimeout(() => {
      setLoading(true);
      api<{ items: Signal[]; total: number }>(
        `/ieds/${iedId}/signals?${buildSignalQuery({ q: query, reportRef, offset,
          kinds: allowedKinds, fc: allowedFc, writableOnly, pointKind })}`)
        .then(data => { if (!cancelled) { setItems(data.items); setTotal(data.total); setError(''); } })
        .catch(reason => { if (!cancelled) setError(reason.message); })
        .finally(() => { if (!cancelled) setLoading(false); });
    }, 150);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [iedId, query, reportRef, offset, pointKind, allowedKinds?.join(','), allowedFc, writableOnly]);
  const chosenKey = chosen ? signalKey(chosen.ied_id, chosen.ref) : '';
  const binding = (signal: Signal): SignalBinding => ({ ied_id: iedId,
    report_ref: reportRef === '__all__' ? null : reportRef, ref: signal.ref });
  const open = items.filter(signal => !exclude?.has(signalKey(iedId, signal.ref)));
  return <div className="signal-search-picker">
    {!fixedIedId && <label>IED<select value={iedId} disabled={disabled}
      onChange={e => change({ iedId: e.target.value, query: '', reportRef: '__all__' })}>
      <option value="">Choose an IED</option>
      {ieds.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}
    </select></label>}
    {ied && <>
      <div className="search-field signal-search-query"><Search size={14} />
        <input aria-label="Search signals" placeholder="Search signals…" value={query}
          disabled={disabled} onChange={e => change({ query: e.target.value })} /></div>
      <div className="signal-search-filters">
        <select aria-label="Report" value={reportRef} disabled={disabled}
          onChange={e => change({ reportRef: e.target.value })}>
          <option value="__all__">All reports</option>
          {ied.reports.map(report => <option key={report.ref} value={report.ref}>{report.ref}</option>)}
        </select>
        <KindSelect value={pointKind} disabled={disabled} onChange={next => change({ pointKind: next })} />
      </div>
    </>}
    {error && <p role="alert">{error}</p>}
    {ied && <><div className="signal-search-list" aria-busy={loading}>
      {loading && !items.length && <Spinner label="Loading signals…" />}
      {items.map(signal => {
        const key = signalKey(iedId, signal.ref);
        const taken = exclude?.has(key);
        const triggerable = signal.writable && signal.fc !== 'CF';
        const kind = kindOfSignal(signal, classOf, ied?.controls);
        return <div className={`signal-search-row kind-edge${key === chosenKey ? ' chosen' : ''}`} key={signal.ref} {...kindRow(kind)}>
          <KindBadge kind={kind} raw={signal.kind} />
          <span className="signal-search-text" title={signal.ref}>
            <strong>{signal.description || shortRef(signal.ref, ied?.name)}</strong>
            <small>{signal.description ? `${shortRef(signal.ref, ied?.name)} · ` : ''}{signal.fc}
              {' · '}{triggerable ? 'can be changed' : 'display only'}</small></span>
          {key === chosenKey ? <span className="badge live">Chosen ✓</span> :
            taken ? <span className="badge live">Added ✓</span> :
            <button className={triggerable && !chosenKey ? 'primary' : ''} disabled={disabled || loading}
              title={chosenKey ? 'Use this point instead' : triggerable ? 'Choose this signal' : 'Choose as display-only'}
              onClick={() => choose(binding(signal), signal)}>
              {chosenKey ? 'Use this' : chooseLabel}</button>}
        </div>;
      })}
      {!loading && !items.length && <p className="small muted">No signals match.</p>}
    </div>
    {chooseAll && open.length > 1 && <button type="button" className="signal-search-all" disabled={disabled || loading}
      title="Adds every signal listed above that is not added yet"
      onClick={() => chooseAll(open.map(signal => ({ binding: binding(signal), signal })))}>
      {`${chooseLabel} all ${open.length} shown`}</button>}
    <div className="row small muted signal-search-pages">
      <span>{total ? offset + 1 : 0}–{Math.min(offset + 50, total)} of {total}</span>
      {loading && !!items.length && <Spinner inline size={14} label="Loading…" />}
      <span style={{ flex: 1 }} />
      <button type="button" className="ghost" disabled={loading || offset === 0}
        onClick={() => setView({ ...view, offset: offset - 50 })}>Previous</button>
      <button type="button" className="ghost" disabled={loading || offset + 50 >= total}
        onClick={() => setView({ ...view, offset: offset + 50 })}>Next</button></div></>}
  </div>;
}
