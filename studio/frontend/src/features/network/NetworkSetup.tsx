import { Check, RefreshCw, X } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { Adapter, ApplyResult, PlanPreview } from '../../api/network';
import { hint } from '../../copy/hints';
import { Disclosure } from '../../ui/Disclosure';
import { Hint } from '../../ui/Hint';
import { AddressTable, type Manual } from './AddressTable';
import { ResultStep, ReviewStep } from './ReviewStep';
import { adapterLabel, changeSentence, usableAdapters } from './setupWords';

/** One page, one button (change 0098): pick the network card, check one address per published IED, apply.
 *  Safety is unchanged: the exact Windows changes are listed, Windows asks once, and Undo/restart removes them. */
export function NetworkSetup({ close, finished }: { close: () => void; finished: () => void }) {
  const [adapters, setAdapters] = useState<Adapter[] | null>(null);
  const [guid, setGuid] = useState('');
  const [manual, setManual] = useState<Manual>({});
  const [preview, setPreview] = useState<PlanPreview | null>(null);
  const [checking, setChecking] = useState(false);
  const [applying, setApplying] = useState(false);
  const [result, setResult] = useState<ApplyResult | null>(null);
  const [error, setError] = useState('');
  const load = useCallback((refresh: boolean) => {
    api<{ adapters: Adapter[] }>(`/network/adapters${refresh ? '?refresh=1' : ''}`)
      .then(answer => setAdapters(answer.adapters)).catch(reason => setError(reason.message));
  }, []);
  useEffect(() => load(false), [load]);
  // A plugged or unplugged cable shows up by itself until a network card is chosen.
  useEffect(() => {
    if (guid || result) return;
    const timer = window.setInterval(() => load(true), 3000);
    return () => window.clearInterval(timer);
  }, [guid, result, load]);
  // Check the addresses again by itself shortly after the card or a typed address changes.
  useEffect(() => {
    if (!guid || result) return;
    const timer = window.setTimeout(() => {
      setChecking(true); setError('');
      api<PlanPreview>('/network/plan/preview', 'POST', { adapter_guid: guid, strategy: 'scd', manual })
        .then(setPreview).catch(reason => { setPreview(null); setError(reason.message); })
        .finally(() => setChecking(false));
    }, 500);
    return () => window.clearTimeout(timer);
  }, [guid, manual, result]);
  const apply = async () => {
    if (!preview) return;
    setApplying(true); setError('');
    try {
      setResult(await api<ApplyResult>('/network/plan/apply', 'POST', { fingerprint: preview.fingerprint, confirm: true }));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally { setApplying(false); }
  };
  const usable = usableAdapters(adapters);
  const count = preview?.rows.length ?? 0;
  if (result) return <div className="card"><div className="guide-section">
    <ResultStep result={result} />
    <div className="row"><button className="primary" onClick={finished}>Done</button></div>
  </div></div>;
  return <div className="card"><div className="guide-section">
    <h2 style={{ margin: 0 }}>Put your published IEDs on a network</h2>
    <div className="row" style={{ alignItems: 'center' }}>
      <label className="row" style={{ gap: 8 }}><strong style={{ whiteSpace: 'nowrap' }}>Network card</strong>
        <select value={guid} onChange={event => { setGuid(event.target.value); setManual({}); setPreview(null); }}
          aria-label="Network card">
          <option value="">Choose the card plugged into the IED or SCADA switch…</option>
          {usable.map(adapter => <option key={adapter.guid} value={adapter.guid}>{adapterLabel(adapter)}</option>)}
        </select></label>
      <button onClick={() => load(true)} aria-label="Refresh the network cards"><RefreshCw size={14} />Refresh</button>
    </div>
    {!guid && <Hint id="network.adapter.pick" compact />}
    {!guid && <Hint id="network.adapter.virtual" compact />}
    {usable.find(item => item.guid === guid)?.default_route && <Hint id="network.adapter.default_route" compact />}
    {adapters && !usable.length && <p className="muted small">No suitable network card found. Plug in a USB
      Ethernet adapter or enable a virtual adapter; this list updates by itself.</p>}
    {error && <div className="error-banner" role="alert" style={{ borderRadius: 6 }}><span>{error}</span></div>}
    {checking && !preview && <p className="small muted">Checking each address, including whether another device
      already answers on it…</p>}
    {preview && <AddressTable preview={preview} manual={manual} setManual={setManual} />}
    {preview && !preview.ready && <Hint id="network.address.fix" compact />}
    {preview && <p className="small" style={{ margin: 0 }}>{changeSentence(preview)}</p>}
    {preview && <Disclosure title="Show the exact Windows changes"><ReviewStep preview={preview} /></Disclosure>}
    <div className="row">
      <button className="primary" disabled={!preview?.ready || checking || applying} onClick={apply}>
        <Check size={16} />{applying ? 'Waiting for Windows approval…'
          : `Put ${count || ''} ${count === 1 ? 'IED' : 'IEDs'} on ${preview ? `“${preview.adapter.name}”` : 'this card'}`}</button>
      <button onClick={close} disabled={applying}><X size={14} />Cancel</button>
    </div>
    {applying && <p className="small muted" style={{ margin: 0 }}>{hint('network.approving')}</p>}
  </div></div>;
}
