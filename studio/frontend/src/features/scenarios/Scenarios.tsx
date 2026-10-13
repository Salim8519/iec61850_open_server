import { useEffect, useState } from 'react';
import { Activity, Play } from 'lucide-react';
import type { Ied, Signal, Scenario } from '../../api/types';
import { api } from '../../api/client';
import { SequenceEditor } from './SequenceEditor';
import { CsvPlayback } from './CsvPlayback';

export function Scenarios({ ieds, scenarios, running, perform }: {
  ieds: Ied[]; scenarios: Scenario[]; running: boolean;
  perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const [iedId, setIedId] = useState(ieds[0]?.id || '');
  const [signals, setSignals] = useState<Signal[]>([]);
  const [reference, setReference] = useState('');
  const [mode, setMode] = useState('sine');
  const [offset, setOffset] = useState(100);
  const [amplitude, setAmplitude] = useState(20);
  const [period, setPeriod] = useState(10);
  const [message, setMessage] = useState('');
  const [query, setQuery] = useState('');
  useEffect(() => { if (!ieds.some(ied => ied.id === iedId)) setIedId(ieds[0]?.id || ''); }, [ieds, iedId]);
  useEffect(() => {
    let active = true;
    if (!iedId) { setSignals([]); setReference(''); return; }
    const timer = setTimeout(() => api<{ items: Signal[] }>(`/ieds/${iedId}/signals?fc=MX&limit=250&q=${encodeURIComponent(query)}`)
      .then(result => {
        if (!active) return;
        const numeric = result.items.filter(item => item.kind.startsWith('FLOAT') || item.kind.startsWith('INT'));
        setSignals(numeric); setReference(numeric[0]?.ref || '');
      }).catch(reason => { if (active) setMessage(reason.message); }), 200);
    return () => { active = false; clearTimeout(timer); };
  }, [iedId, query]);
  return <div className="main-view"><div className="view-heading"><div><h1>Scenarios</h1>
    <p className="muted small">Drive measurements automatically and observe the real MMS values.</p></div><Activity size={22} /></div>
    <div className="content-page" style={{ paddingTop: 0 }}><div className="card" style={{ maxWidth: 850 }}>
      <div className="card-heading"><h2>Automatic measurement source</h2><span className="badge">10 Hz</span></div>
      <div className="stack" style={{ padding: 24 }}>
        <label>Search measurements<input value={query} onChange={event => setQuery(event.target.value)} placeholder="Filter exact references" /></label>
        <div className="generator-grid" style={{ gridTemplateColumns: '1fr 2fr' }}>
          <label>IED<select value={iedId} onChange={e => setIedId(e.target.value)}><option value="">Choose IED</option>
            {ieds.map(ied => <option key={ied.id} value={ied.id}>{ied.name}</option>)}</select></label>
          <label>Measurement<select value={reference} onChange={e => setReference(e.target.value)}>
            {signals.map(signal => <option key={signal.ref}>{signal.ref}</option>)}</select></label>
        </div>
        <div className="generator-grid">
          <label>Source<select value={mode} onChange={e => setMode(e.target.value)}>
            {['sine', 'ramp', 'step', 'constant', 'noise', 'manual', 'csv'].map(item => <option key={item}>{item}</option>)}</select></label>
          {mode !== 'csv' && <><label>Offset<input type="number" value={offset} onChange={e => setOffset(Number(e.target.value))} /></label>
          <label>Amplitude<input type="number" value={amplitude} onChange={e => setAmplitude(Number(e.target.value))} /></label>
          <label>Period (seconds)<input type="number" min="0.1" step="0.1" value={period} onChange={e => setPeriod(Number(e.target.value))} /></label></>}
        </div>
        {mode === 'csv' ? <CsvPlayback iedId={iedId} reference={reference} perform={perform} /> :
        <div className="row"><button className="primary" disabled={!reference} onClick={() => perform(async () => {
          await api('/signals/source', 'POST', { ied_id: iedId, ref: reference, spec: { mode, offset, amplitude, period } });
          setMessage('Source configured. It runs while automation is active.');
        })}><Play size={14} />Apply source</button><span className="small muted">{message}</span></div>}
        <p className="small muted" style={{ marginBottom: 0 }}>Manual editing in IEDs & Signals pauses this source for that point. Apply the source again to resume it.</p>
      </div>
    </div><SequenceEditor ieds={ieds} scenarios={scenarios} running={running} perform={perform} /></div>
  </div>;
}
