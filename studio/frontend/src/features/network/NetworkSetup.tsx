import { Check, RefreshCw, X } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../../api/client';
import { liveFeed } from '../../api/polls';
import type { Adapter, ApplyResult, PlanPreview } from '../../api/network';
import { hint } from '../../copy/hints';
import { Disclosure } from '../../ui/Disclosure';
import { Hint } from '../../ui/Hint';
import { SpinIcon, Spinner } from '../../ui/Spinner';
import { AddressTable, type Manual } from './AddressTable';
import { ResultStep, ReviewStep } from './ReviewStep';
import { adapterLabel, changeSentence, putLabel, usableAdapters } from './setupWords';

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
  // Only a Refresh the user pressed shows a turning icon; the quiet 3 s re-check does not.
  const [refreshing, setRefreshing] = useState(false);
  const load = useCallback((refresh: boolean, pressed = false) => {
    if (pressed) setRefreshing(true);
    api<{ adapters: Adapter[] }>(`/network/adapters${refresh ? '?refresh=1' : ''}`)
      .then(answer => setAdapters(answer.adapters)).catch(reason => setError(reason.message))
      .finally(() => { if (pressed) setRefreshing(false); });
  }, []);
  useEffect(() => load(false), [load]);
  // A plugged or unplugged cable shows up by itself until a network card is chosen.
  useEffect(() => {
    if (guid || result) return undefined;
    // Shared live feed (speed lane F): starts after the first 3 s, as before; no overlap, no
    // re-render while the adapters stay the same, paused while the window is hidden.
    let stop = () => {};
    const start = window.setTimeout(() => {
      stop = liveFeed.subscribe<{ adapters: Adapter[] }>('/network/adapters?refresh=1', 3000,
        answer => setAdapters(answer.adapters), reason => setError(reason.message));
    }, 3000);
    return () => { window.clearTimeout(start); stop(); };
  }, [guid, result]);
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
        <select value={guid} disabled={!adapters} onChange={event => { setGuid(event.target.value); setManual({}); setPreview(null); }}
          aria-label="Network card">
          <option value="">{adapters ? 'Choose the card plugged into the IED or SCADA switch…' : 'Looking for network cards…'}</option>
          {usable.map(adapter => <option key={adapter.guid} value={adapter.guid}>{adapterLabel(adapter)}</option>)}
        </select></label>
      <button onClick={() => load(true, true)} disabled={refreshing} aria-label="Refresh the network cards">
        {refreshing ? <SpinIcon size={14} /> : <RefreshCw size={14} />}{refreshing ? 'Refreshing…' : 'Refresh'}</button>
    </div>
    {!adapters && !error && <Spinner label="Looking for network cards…"
      slowHint="Still working. Windows can take a few seconds to list its network cards." />}
    {!guid && <Hint id="network.adapter.pick" compact />}
    {!guid && <Hint id="network.adapter.virtual" compact />}
    {usable.find(item => item.guid === guid)?.default_route && <Hint id="network.adapter.default_route" compact />}
    {adapters && !usable.length && <p className="muted small">No suitable network card found. Plug in a USB
      Ethernet adapter or enable a virtual adapter; this list updates by itself.</p>}
    {error && <div className="error-banner" role="alert" style={{ borderRadius: 6 }}><span>{error}</span></div>}
    {checking && !preview && <Spinner label="Checking each address, including whether another device already answers on it…"
      slowHint="Still checking. Addresses nobody answers on take a moment to confirm." />}
    {checking && preview && <Spinner inline size={14} label="Checking again…" />}
    {preview && <AddressTable preview={preview} manual={manual} setManual={setManual} />}
    {preview && !preview.ready && <Hint id="network.address.fix" compact />}
    {preview && <p className="small" style={{ margin: 0 }}>{changeSentence(preview)}</p>}
    {preview && <Disclosure title="Show the exact Windows changes"><ReviewStep preview={preview} /></Disclosure>}
    <div className="row">
      <button className="primary" disabled={!preview?.ready || checking || applying} onClick={apply}>
        {applying ? <SpinIcon /> : <Check size={16} />}{applying ? 'Waiting for Windows approval…'
          : putLabel(count, preview?.adapter.name)}</button>
      <button onClick={close} disabled={applying}><X size={14} />Cancel</button>
    </div>
    {applying && <p className="small muted" style={{ margin: 0 }}>{hint('network.approving')}</p>}
  </div></div>;
}
