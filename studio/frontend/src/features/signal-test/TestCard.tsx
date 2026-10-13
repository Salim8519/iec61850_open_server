import { useEffect, useState } from 'react';
import type { Signal, StudioEvent, TestControl } from '../../api/types';
import { SignalValueEditor } from '../../components/SignalValueEditor';
import { api } from '../../api/client';

export type TestItem = TestControl & { signal: Signal; automatic: boolean; overridden: boolean };
type Props = { item: TestItem; running: boolean; values: Record<string, unknown>; iedName: string;
  equipment: string[]; events: StudioEvent[]; first: boolean; last: boolean;
  edit: (entry: TestControl) => void; remove: () => void; move: (direction: number) => void;
  perform: (task: () => Promise<unknown>) => Promise<void> };

export function TestCard({ item, running, values, iedName, equipment, events, first, last, edit, remove, move, perform }: Props) {
  const [label, setLabel] = useState(item.label);
  const [duration, setDuration] = useState(String(item.duration_ms));
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => { setLabel(item.label); setDuration(String(item.duration_ms)); }, [item.label, item.duration_ms]);
  const signal = item.signal;
  const value = values[`${item.ied_id}:${item.ref}`] ?? signal.value;
  const quality = values[`${item.ied_id}:${signal.quality_ref}`] ?? signal.quality;
  const timestamp = Number(values[`${item.ied_id}:${signal.timestamp_ref}`] ?? signal.timestamp);
  const latest = [...events].reverse().find(event => event.ied_id === item.ied_id && event.ref === item.ref && event.kind.startsWith('pulse_'));
  const action = async (path: string, extra: Record<string, unknown>) => {
    setError(''); setBusy(true);
    await perform(async () => {
      try { return await api(path, 'POST', { ied_id: item.ied_id, ref: item.ref, ...extra }); }
      catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); throw reason; }
    });
    setBusy(false);
  };
  return <section className="card test-card">
    <div className="row"><h2>{item.label}</h2><span className="badge">{signal.kind} · {signal.fc}</span></div>
    <p className="small muted">{iedName} · {item.report_ref || 'All signals / not report-filtered'}</p>
    <code className="reference-block">{item.ref}</code>
    <p className="small muted">SLD: {equipment.length ? equipment.join(', ') : 'No symbol needed — independent test control'}</p>
    {!running && <details><summary>Edit saved control</summary><div className="stack">
      <label>Label<input maxLength={100} value={label} onChange={e => setLabel(e.target.value)} /></label>
      {signal.kind === 'BOOLEAN' && <>
        <label>Pulse duration (ms)<input type="number" min={100} max={60000} step={100} value={duration} onChange={e => setDuration(e.target.value)} /></label>
        <label>Pulse active value<select value={String(item.active_value)} onChange={e => edit({ ...item, active_value: e.target.value === 'true' })}>
          <option value="true">True</option><option value="false">False</option></select></label>
      </>}
      <div className="row"><button disabled={!label.trim() || !Number.isInteger(Number(duration)) || Number(duration) < 100 || Number(duration) > 60000}
        onClick={() => edit({ ...item, label, duration_ms: Number(duration) })}>Save control</button>
        <button disabled={first} onClick={() => move(-1)}>Move up</button><button disabled={last} onClick={() => move(1)}>Move down</button>
        <button className="danger-outline" onClick={remove}>Remove control</button></div>
    </div></details>}
    <div className="test-value"><SignalValueEditor signal={signal} value={value} disabled={!running || busy}
      update={next => action('/signals/update', { value: next })} /></div>
    <p className="small">Quality: {signal.quality_ref ? Number(quality) === 0 ? 'Good' : `Flags ${quality}` : 'Not present'} ·
      Time: {timestamp ? new Date(timestamp).toLocaleTimeString() : 'Not set'}</p>
    <div className="row">
      {signal.kind === 'BOOLEAN' && <button disabled={!running || busy} onClick={() => action('/signals/pulse', {
        active_value: item.active_value, duration_ms: item.duration_ms,
      })}>Pulse {String(item.active_value)} · {item.duration_ms} ms</button>}
      {running && item.automatic && <><span className="badge">{item.overridden ? 'Manual override' : 'Automatic source'}</span>
        <button disabled={busy || !item.overridden} onClick={() => action('/signals/release-override', {})}>Resume automatic source</button></>}
    </div>
    {latest && running && <p className="small" role="status">{latest.kind.replace('pulse_', 'Pulse ').replaceAll('_', ' ')}{latest.message ? `: ${latest.message}` : ''}</p>}
    {!running && <p className="small muted">Run the IED to change live values. Saved controls never fire on startup.</p>}
    {error && <p role="alert">{error}</p>}
  </section>;
}
