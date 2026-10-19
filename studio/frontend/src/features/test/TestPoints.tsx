import { ArrowDown, ArrowUp, ListChecks, Plus, Trash2, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { Project, Signal, TestControl } from '../../api/types';
import { SignalSearchPicker } from '../../components/SignalSearchPicker';
import '../../components/signal-search-picker.css';
import { shortRef } from '../../equipment/relayDisplay';
import { Disclosure } from '../../ui/Disclosure';
import { EmptyState } from '../../ui/EmptyState';
import { Spinner } from '../../ui/Spinner';
import { ValueControl } from './ValueControl';

type Item = TestControl & { signal: Signal; automatic: boolean; overridden: boolean };

export function TestPoints({ project, values, perform, prepare }: {
  project: Project; values: Record<string, unknown>;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
}) {
  const [items, setItems] = useState<Item[]>([]);
  const [adding, setAdding] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const running = project.runtime !== 'stopped';
  const entries = project.test_panel || [];
  useEffect(() => {
    let cancelled = false;
    const load = () => api<{ items: Item[] }>('/project/test-panel')
      .then(result => { if (!cancelled) { setItems(result.items); setLoaded(true); } });
    load();
    const timer = running ? setInterval(load, 1000) : undefined;
    return () => { cancelled = true; clearInterval(timer); };
  }, [project.test_panel, running]);
  const save = (next: TestControl[]) => perform(async () => { await prepare(); await api('/project/test-panel', 'PUT', next); });
  const move = (index: number, direction: number) => {
    const next = [...entries]; [next[index], next[index + direction]] = [next[index + direction], next[index]]; save(next);
  };
  const edit = (entry: TestControl) => save(entries.map(item => item.id === entry.id ? entry : item));
  return <div className="stack">
    <div className="row" style={{ justifyContent: 'space-between' }}>
      <p className="muted" style={{ margin: 0 }}>Your saved alarms, statuses and measurements. Press Run, then change them here.</p>
      {(adding || !!items.length) && <button className={adding ? '' : 'primary'} disabled={running} onClick={() => setAdding(!adding)}>
        {adding ? <><X size={16} />Close</> : <><Plus size={16} />Add test point</>}</button>}
    </div>
    {adding && <div className="card"><div className="card-body"><SignalSearchPicker ieds={project.ieds} writableOnly chooseLabel="Add" memoryKey="test-points"
      exclude={new Set(entries.map(item => `${item.ied_id}:${item.ref}`))}
      choose={(binding, signal) => save([...entries, { id: crypto.randomUUID(), label: (signal.description || signal.ref).slice(0, 100),
        ...binding, duration_ms: 1000, active_value: true }])} /></div></div>}
    {!loaded && !items.length && <div className="card"><div className="card-body"><Spinner label="Loading your test points…" /></div></div>}
    {loaded && !items.length && !adding && <div className="card"><EmptyState icon={<ListChecks size={24} />} title="No test points yet"
      hintId="test.points.empty" action={<button className="primary" disabled={running} onClick={() => setAdding(true)}>
        <Plus size={16} />Add test point</button>} /></div>}
    {!!items.length && <div className="card"><table><thead><tr><th>Name</th><th>Value</th><th style={{ width: 120 }} /></tr></thead>
      <tbody>{items.map((item, index) => {
        const ied = project.ieds.find(entry => entry.id === item.ied_id);
        return <tr key={item.id}><td style={{ minWidth: 240 }}><strong>{item.label}</strong>
          <div className="small muted">{ied?.name || 'Missing device'} · {shortRef(item.ref, ied?.name)}</div>
          {!running && <Disclosure title="Edit"><label>Name<input maxLength={100} defaultValue={item.label}
            onBlur={event => event.target.value.trim() && event.target.value !== item.label && edit({ ...item, label: event.target.value })} /></label>
            {item.signal.kind === 'BOOLEAN' && <div className="form-grid">
              <label>Pulse value<select value={String(item.active_value)} onChange={event => edit({ ...item, active_value: event.target.value === 'true' })}>
                <option value="true">True</option><option value="false">False</option></select></label>
              <label>Pulse length (ms)<input type="number" min={100} max={60000} step={100} defaultValue={item.duration_ms}
                onBlur={event => Number(event.target.value) !== item.duration_ms && edit({ ...item, duration_ms: Number(event.target.value) })} /></label>
            </div>}</Disclosure>}</td>
          <td><ValueControl point={item} signal={item.signal} running={running} perform={perform}
            value={values[`${item.ied_id}:${item.ref}`] ?? item.signal.value} segmented
            pulse={{ active: item.active_value, durationMs: item.duration_ms }} generator={item.automatic} /></td>
          <td><div className="row" style={{ justifyContent: 'flex-end', gap: 2 }}>
            <button className="icon ghost" aria-label={`Move ${item.label} up`} disabled={running || index === 0} onClick={() => move(index, -1)}><ArrowUp size={16} /></button>
            <button className="icon ghost" aria-label={`Move ${item.label} down`} disabled={running || index === items.length - 1} onClick={() => move(index, 1)}><ArrowDown size={16} /></button>
            <button className="icon ghost" aria-label={`Remove ${item.label}`} disabled={running}
              onClick={() => save(entries.filter(entry => entry.id !== item.id))}><Trash2 size={16} /></button></div></td></tr>;
      })}</tbody></table></div>}
  </div>;
}
