import { Search } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { GatewayPoint, PointKind, Protocol } from '../../../../api/gateway';
import { addApi, type Candidate } from '../../../../api/gatewayAdd';
import { ADD_COPY } from '../../../../copy/gatewayAdd';
import { kindOfGatewayPoint } from '../../../../points/pointKind';
import { KindBadge, kindRow } from '../../../../ui/KindBadge';
import { Spinner } from '../../../../ui/Spinner';
import { tickShown, toggle } from './addPlan';

/** Step 3 of "From an IED": only this IED's signals or commands that fit the type (the engine decides),
 *  searchable, with a tick per row. Points already in the list are shown as such and cannot be ticked. */
export function CandidateList({ protocol, ied, kind, rows, ticked, setTicked, fail }: {
  protocol: Protocol; ied: string; kind: PointKind; rows: GatewayPoint[]; ticked: string[];
  setTicked: (next: string[]) => void; fail: (message: string) => void;
}) {
  const [query, setQuery] = useState('');
  const [items, setItems] = useState<Candidate[] | null>(null);
  const [total, setTotal] = useState(0);
  useEffect(() => setItems(null), [ied, kind]); // a new IED or type shows the spinner, never the old list
  useEffect(() => {
    let live = true;
    const timer = setTimeout(() => addApi.candidates(protocol, ied, kind, query, rows)
      .then(answer => { if (live) { setItems(answer.items); setTotal(answer.total); } })
      .catch(error => fail(error.message)), query ? 200 : 0);
    return () => { live = false; clearTimeout(timer); };
  }, [protocol, ied, kind, query]);
  const badge = kindOfGatewayPoint(kind);
  return <div className="candidate-list">
    <div className="row candidate-tools">
      <div className="search-field"><Search size={14} aria-hidden />
        <input aria-label={ADD_COPY.searchSignals} placeholder={ADD_COPY.searchSignals} value={query}
          onChange={event => setQuery(event.target.value)} /></div>
      <button type="button" className="ghost" disabled={!items?.length} onClick={() => setTicked(tickShown(ticked, items ?? []))}>
        {ADD_COPY.tickShown}</button>
      <button type="button" className="ghost" disabled={!ticked.length} onClick={() => setTicked([])}>{ADD_COPY.untickAll}</button>
    </div>
    <div className="candidate-rows" aria-busy={items === null}>
      {items === null && <Spinner label={ADD_COPY.loadingSignals} />}
      {items?.map(item => <label key={item.ref} className={`candidate-row kind-edge${item.taken ? ' taken' : ''}`} {...kindRow(badge)}>
        <input type="checkbox" checked={item.taken || ticked.includes(item.ref)} disabled={item.taken}
          onChange={() => setTicked(toggle(ticked, item.ref))} />
        <KindBadge kind={badge} raw={item.cdc || item.value_kind} />
        <span className="candidate-text" title={item.ref}><strong>{item.name}</strong>
          <small>{item.ref}{item.cdc ? ` · ${item.cdc}` : ` · ${item.fc} ${item.value_kind}`}</small></span>
        {item.suggested && !item.cdc && <span className="badge" title={ADD_COPY.suggestedHint}>✓ {ADD_COPY.suggested}</span>}
        {item.taken && <span className="badge live">✓ {ADD_COPY.inList}</span>}
      </label>)}
      {items && !items.length && <p className="small muted">{query ? ADD_COPY.noSignalMatch : ADD_COPY.noSignals}</p>}
    </div>
    {items && total > items.length && <p className="small muted">{ADD_COPY.more(items.length, total)}</p>}
  </div>;
}
