import { useEffect, useState } from 'react';
import type { EquipmentData, Project, Signal } from '../../api/types';
import { api } from '../../api/client';
import { SignalValueEditor } from '../../components/SignalValueEditor';
import { CsvPlayback } from '../scenarios/CsvPlayback';

export function MeasurementSourceEditor({ data, project, values, perform }: {
  data: EquipmentData; project: Project; values: Record<string, unknown>;
  perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const binding = data.state_mapping?.mode === 'direct' ? data.state_mapping.source : null;
  const configured = binding ? project.sources.find(item =>
    item.ied_id === binding.ied_id && item.ref === binding.ref)?.spec : undefined;
  const [signal, setSignal] = useState<Signal | null>(null);
  const [mode, setMode] = useState(configured?.mode || 'manual');
  const [offset, setOffset] = useState(configured?.offset ?? 0);
  const [amplitude, setAmplitude] = useState(configured?.amplitude ?? 1);
  const [period, setPeriod] = useState(configured?.period ?? 10);
  const [message, setMessage] = useState('');
  useEffect(() => {
    setMode(configured?.mode || 'manual'); setOffset(configured?.offset ?? 0);
    setAmplitude(configured?.amplitude ?? 1); setPeriod(configured?.period ?? 10);
  }, [binding?.ied_id, binding?.ref, configured?.mode, configured?.offset,
    configured?.amplitude, configured?.period]);
  useEffect(() => {
    let cancelled = false;
    setSignal(null);
    if (!binding) return;
    api<{ items: Signal[] }>(`/ieds/${binding.ied_id}/signals?ref=${encodeURIComponent(binding.ref)}`)
      .then(result => { if (!cancelled) setSignal(result.items[0] || null); })
      .catch(reason => { if (!cancelled) setMessage(reason.message); });
    return () => { cancelled = true; };
  }, [binding?.ied_id, binding?.ref]);
  if (!binding) return <p className="small muted">Assign a numeric MX point to configure its source.</p>;
  const running = project.runtime !== 'stopped';
  const value = signal ? values[`${binding.ied_id}:${binding.ref}`] ?? signal.value : undefined;
  return <div className="stack"><label>Mode<select value={mode} onChange={event => setMode(event.target.value)}>
      {['manual', 'constant', 'ramp', 'sine', 'step', 'noise', 'csv'].map(item =>
        <option key={item} value={item}>{item[0].toUpperCase() + item.slice(1)}</option>)}</select></label>
    {mode === 'csv' ? <CsvPlayback iedId={binding.ied_id} reference={binding.ref} perform={perform} /> : <>
      {mode !== 'manual' && <><label>Offset<input type="number" step="any" value={offset}
        onChange={event => setOffset(Number(event.target.value))} /></label>
      <label>Amplitude<input type="number" step="any" value={amplitude}
        onChange={event => setAmplitude(Number(event.target.value))} /></label>
      <label>Period (seconds)<input type="number" min="0.1" step="0.1" value={period}
        onChange={event => setPeriod(Number(event.target.value))} /></label></>}
      <button className="primary" onClick={() => perform(async () => {
        await api('/signals/source', 'POST', { ied_id: binding.ied_id, ref: binding.ref,
          spec: { mode, offset, amplitude, period } });
        setMessage(mode === 'manual' ? 'Manual control selected.' : 'Automatic source configured.');
      })}>Apply {mode} source</button>
    </>}
    {running && signal && <label>Live value<SignalValueEditor signal={signal} value={value}
      disabled={false} variant="segmented" update={next => perform(() => api('/signals/update', 'POST', {
        ied_id: binding.ied_id, ref: binding.ref, value: next }))} /></label>}
    {message && <p className="small" role="status">{message}</p>}
  </div>;
}
