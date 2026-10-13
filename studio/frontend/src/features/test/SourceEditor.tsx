import { useEffect, useState } from 'react';
import { Play } from 'lucide-react';
import { api } from '../../api/client';
import type { Project, Signal } from '../../api/types';
import { CsvPlayback } from './CsvPlayback';
import { ValueControl } from './ValueControl';
import { configureSource, type Point } from './signalActions';
import { MODES, NUMERIC } from './sourceModes';
import { shortRef } from '../../equipment/relayDisplay';

/** The one generator editor (UI-06): SOURCE symbols and the Test > Generators list use it. */
export function SourceEditor({ point: fixed, project, values, perform, done }: {
  point?: Point | null; project: Project; values: Record<string, unknown>;
  perform: (task: () => Promise<unknown>) => Promise<void>; done?: () => void;
}) {
  const [iedId, setIedId] = useState(fixed?.ied_id || project.ieds[0]?.id || '');
  const [ref, setRef] = useState(fixed?.ref || '');
  const [query, setQuery] = useState('');
  const [choices, setChoices] = useState<Signal[]>([]);
  const [signal, setSignal] = useState<Signal | null>(null);
  const point = fixed || (iedId && ref ? { ied_id: iedId, ref } : null);
  const configured = point ? project.sources.find(item => item.ied_id === point.ied_id && item.ref === point.ref)?.spec : undefined;
  const [mode, setMode] = useState(configured?.mode || 'sine');
  const [offset, setOffset] = useState(configured?.offset ?? 100);
  const [amplitude, setAmplitude] = useState(configured?.amplitude ?? 10);
  const [period, setPeriod] = useState(configured?.period ?? 10);
  const [message, setMessage] = useState('');
  useEffect(() => {
    setMode(configured?.mode || (fixed ? 'manual' : 'sine')); setOffset(configured?.offset ?? (fixed ? 0 : 100));
    setAmplitude(configured?.amplitude ?? 10); setPeriod(configured?.period ?? 10);
  }, [point?.ied_id, point?.ref, configured?.mode]);
  useEffect(() => {
    if (fixed || !iedId) return;
    let active = true;
    const timer = setTimeout(() => api<{ items: Signal[] }>(`/ieds/${iedId}/signals?fc=MX&limit=200&q=${encodeURIComponent(query)}`)
      .then(result => { if (active) setChoices(result.items.filter(item => NUMERIC(item.kind))); }), 200);
    return () => { active = false; clearTimeout(timer); };
  }, [fixed, iedId, query]);
  useEffect(() => {
    setSignal(null);
    if (!point) return;
    let active = true;
    api<{ items: Signal[] }>(`/ieds/${point.ied_id}/signals?ref=${encodeURIComponent(point.ref)}`)
      .then(result => { if (active) setSignal(result.items[0] || null); });
    return () => { active = false; };
  }, [point?.ied_id, point?.ref]);
  const running = project.runtime !== 'stopped';
  const info = MODES.find(item => item.value === mode) || MODES[0];
  return <div className="stack">
    {!fixed && <div className="form-grid" style={{ gridTemplateColumns: 'minmax(160px, 1fr) minmax(160px, 1fr) minmax(260px, 2fr)' }}>
      <label>Device<select value={iedId} onChange={event => { setIedId(event.target.value); setRef(''); }}>
        {project.ieds.map(ied => <option key={ied.id} value={ied.id}>{ied.name}</option>)}</select></label>
      <label>Search measurements<input value={query} placeholder="e.g. MMXU1 or TotW" onChange={event => setQuery(event.target.value)} /></label>
      <label>Measurement<select value={ref} onChange={event => setRef(event.target.value)}>
        <option value="">Choose a measurement</option>
        {choices.map(item => <option key={item.ref} value={item.ref}>
          {item.description ? `${item.description} · ${shortRef(item.ref, project.ieds.find(ied => ied.id === iedId)?.name)}` : item.ref}</option>)}</select></label>
    </div>}
    {fixed && !signal && <p className="small muted">Connect this symbol to a measurement first.</p>}
    {point && <>
      <label style={{ maxWidth: 420 }}>How the value changes<select value={mode} onChange={event => setMode(event.target.value)}>
        {MODES.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>
      <p className="small muted" style={{ margin: 0 }}>{info.explain}</p>
      {mode === 'csv' ? <CsvPlayback iedId={point.ied_id} reference={point.ref} perform={perform} /> : <>
        {mode !== 'manual' && <div className="generator-grid">
          <label>{mode === 'constant' ? 'Value' : 'Middle value'}<input type="number" step="any" value={offset}
            onChange={event => setOffset(Number(event.target.value))} /></label>
          {mode !== 'constant' && <><label>Swing (±)<input type="number" step="any" value={amplitude}
            onChange={event => setAmplitude(Number(event.target.value))} /></label>
          <label>Cycle time (seconds)<input type="number" min="0.1" step="0.1" value={period}
            onChange={event => setPeriod(Number(event.target.value))} /></label></>}</div>}
        <div className="row"><button className="primary" onClick={() => perform(async () => {
          await configureSource(point, { mode, offset, amplitude, period });
          setMessage(mode === 'manual' ? 'You now set this value by hand.' : 'Saved. It runs while the project is running.');
          done?.();
        })}><Play size={15} />Save generator</button><span className="small muted" role="status">{message}</span></div>
      </>}
      {running && signal && <label>Value now<ValueControl point={point} signal={signal} running={running}
        value={values[`${point.ied_id}:${point.ref}`] ?? signal.value} perform={perform}
        generator={!!configured && configured.mode !== 'manual'} /></label>}
    </>}
  </div>;
}
