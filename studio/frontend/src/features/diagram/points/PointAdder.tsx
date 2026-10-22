import { useEffect, useState } from 'react';
import { Search } from 'lucide-react';
import { api } from '../../../api/client';
import type { ExtendedControl, Ied, PanelPoint, PointType, Signal } from '../../../api/types';
import { buildSignalQuery, SIGNAL_PAGE } from '../../../components/signalQuery';
import { KindBadge, kindRow } from '../../../ui/KindBadge';
import { SegmentedControl } from '../../../ui/SegmentedControl';
import { Spinner } from '../../../ui/Spinner';
import { kindOfPoint } from '../../../equipment/pointDisplay';
import { MAX_PANEL_POINTS, newCommandPoint, newSignalPoint, POINT_SIGNAL_KINDS, pointKeys, withIds } from '../../../equipment/pointPanel';
import { shortRef } from '../../../equipment/relayDisplay';
import { commandControls } from '../../../equipment/pointRequests';
import { controlSummary } from '../../../equipment/typedCommand';
import { POINT_COPY } from '../../../copy/pointPanel';

type Choice = { key: string; ref: string; text: string; draft: PanelPoint };
const QUERY: Record<Exclude<PointType, 'command'>, { kinds: string[]; pointKind: string; fc?: string }> = {
  status: { kinds: POINT_SIGNAL_KINDS.status, pointKind: 'SP' },
  double: { kinds: POINT_SIGNAL_KINDS.double, pointKind: 'DP' },
  value: { kinds: POINT_SIGNAL_KINDS.value, pointKind: 'MV', fc: 'MX' },
};
const TYPES = ['status', 'double', 'value', 'command'] as const;
/** Value rows read a measured number (MX) or a whole-number status (ST INT…: tap position, counter). */
type ValueSource = 'MX' | 'ST';
const WHOLE_STATUS = { kinds: POINT_SIGNAL_KINDS.value.filter(kind => kind.startsWith('INT')), fc: 'ST' };
const queryFor = (type: Exclude<PointType, 'command'>, source: ValueSource) =>
  type === 'value' && source === 'ST' ? WHOLE_STATUS : QUERY[type];

/** Add points of one IED: choose a type, search, tick several (across pages and types), press Add N. */
export function PointAdder({ ied, extended, points, add, done }: {
  ied: Ied; extended: readonly ExtendedControl[]; points: PanelPoint[]; add: (fresh: PanelPoint[]) => number; done: () => void;
}) {
  const [type, setType] = useState<PointType>('status');
  const [query, setQuery] = useState('');
  const [offset, setOffset] = useState(0);
  const [signals, setSignals] = useState<Signal[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [ticked, setTicked] = useState<ReadonlyMap<string, Choice>>(new Map());
  const [notice, setNotice] = useState('');
  const [source, setSource] = useState<ValueSource>('MX');
  const held = pointKeys(points);
  useEffect(() => {
    if (type === 'command') { setSignals([]); setTotal(0); return undefined; }
    let cancelled = false;
    const timer = setTimeout(() => {
      setLoading(true);
      api<{ items: Signal[]; total: number }>(`/ieds/${encodeURIComponent(ied.id)}/signals?${buildSignalQuery({ q: query, offset, ...queryFor(type, source) })}`)
        .then(data => { if (!cancelled) { setSignals(data.items); setTotal(data.total); setError(''); } })
        .catch(reason => { if (!cancelled) setError(reason.message); })
        .finally(() => { if (!cancelled) setLoading(false); });
    }, 150);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [ied.id, type, query, offset, source]);
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  const rows: Choice[] = type === 'command'
    ? commandControls(ied, extended).filter(control => words.every(word => control.ref.toLowerCase().includes(word))).map(control => ({
      key: `command:${control.ref}`, ref: control.ref, text: controlSummary(control),
      draft: newCommandPoint(ied.id, control, '', ied.name) }))
    : signals.map(signal => ({ key: `${type}:${signal.ref}`, ref: signal.ref, text: `${signal.fc} · ${signal.kind}`,
      draft: newSignalPoint(type, { ied_id: ied.id, report_ref: null, ref: signal.ref }, signal, '', ied.name) }));
  const open = rows.filter(row => !held.has(row.key));
  const toggle = (row: Choice) => setTicked(current => {
    const next = new Map(current);
    if (next.has(row.key)) next.delete(row.key); else next.set(row.key, row);
    return next;
  });
  const allTicked = open.length > 0 && open.every(row => ticked.has(row.key));
  const tickAll = () => setTicked(current => {
    const next = new Map(current);
    for (const row of open) { if (allTicked) next.delete(row.key); else next.set(row.key, row); }
    return next;
  });
  const addTicked = () => {
    const skipped = add(withIds(points, [...ticked.values()].map(row => row.draft)));
    setNotice(skipped ? POINT_COPY.someSkipped(skipped) : POINT_COPY.added(ticked.size));
    setTicked(new Map());
  };
  const controlFor = (row: Choice) => commandControls(ied, extended).find(control => control.ref === row.ref);
  const kindOf = (row: Choice) => kindOfPoint(row.draft, row.draft.type === 'command' ? controlFor(row) : undefined);
  const room = MAX_PANEL_POINTS - points.length;
  const pageEnd = Math.min(offset + SIGNAL_PAGE, total);
  return <div className="pp-adder">
    <SegmentedControl label="Type of point" value={type} full
      options={TYPES.map(value => ({ value, label: POINT_COPY.typeNames[value], title: POINT_COPY.typeHints[value] }))}
      onChange={next => { setType(next); setOffset(0); }} />
    <p className="small muted pp-adder-hint">{POINT_COPY.typeHints[type]}</p>
    {type === 'value' && <SegmentedControl label="Kind of value" value={source}
      options={[{ value: 'MX', label: POINT_COPY.valueMeasured, title: POINT_COPY.valueMeasuredHint },
        { value: 'ST', label: POINT_COPY.valueWhole, title: POINT_COPY.valueWholeHint }]}
      onChange={next => { setSource(next); setOffset(0); }} />}
    <div className="search-field"><Search size={14} aria-hidden="true" />
      <input aria-label="Search signals" placeholder="Search signals…" value={query}
        onChange={event => { setQuery(event.target.value); setOffset(0); }} /></div>
    {error && <p role="alert" className="relay-error">{error}</p>}
    <div className="pp-adder-list" aria-busy={loading}>
      {loading && !rows.length && <Spinner label="Loading signals…" />}
      {rows.map(row => {
        const taken = held.has(row.key);
        return <label key={row.key} className={`pp-adder-row kind-edge${taken ? ' taken' : ''}`} {...kindRow(kindOf(row))}>
          <input type="checkbox" disabled={taken} checked={taken || ticked.has(row.key)} onChange={() => toggle(row)} />
          <KindBadge kind={kindOf(row)} raw={row.draft.type === 'command' ? controlFor(row)?.cdc : undefined} />
          {row.draft.type === 'command' && <span className="kind-name">{controlFor(row)?.cdc}</span>}
          <span className="pp-adder-text" title={row.ref}><strong>{row.draft.label}</strong>
            <small>{shortRef(row.ref, ied.name)} · {row.text}</small></span>
          {taken && <span className="badge live">Added ✓</span>}
        </label>;
      })}
      {!loading && !rows.length && <p className="small muted">No signals match.</p>}
    </div>
    <div className="pp-adder-foot">
      {type !== 'command' && <span className="small muted">{total ? offset + 1 : 0}–{pageEnd} of {total}</span>}
      {type !== 'command' && <>
        <button className="ghost" disabled={loading || offset === 0} onClick={() => setOffset(offset - SIGNAL_PAGE)}>Previous</button>
        <button className="ghost" disabled={loading || offset + SIGNAL_PAGE >= total} onClick={() => setOffset(offset + SIGNAL_PAGE)}>Next</button></>}
      <span className="pp-spacer" />
      {open.length > 1 && <button className="ghost" onClick={tickAll}>{allTicked ? 'Untick shown' : POINT_COPY.tickAll}</button>}
      <button className="primary" disabled={!ticked.size || room <= 0} onClick={addTicked}>{POINT_COPY.addN(ticked.size)}</button>
      <button onClick={done}>Done</button>
    </div>
    {notice && <p className="small" role="status">{notice}</p>}
    {room <= 0 && <p className="small muted">{POINT_COPY.limit(MAX_PANEL_POINTS)}</p>}
  </div>;
}
