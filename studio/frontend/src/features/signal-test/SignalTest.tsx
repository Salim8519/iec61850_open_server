import { useEffect, useState } from 'react';
import type { Project, StudioEvent, TestControl } from '../../api/types';
import { SignalPicker } from '../../components/SignalPicker';
import { api } from '../../api/client';
import { TestCard, type TestItem } from './TestCard';
import { stateBindings, relayBindings } from '../../equipment/stateBindings';
import './signal-test.css';

export function SignalTest({ project, values, events, perform, prepare }: {
  project: Project; values: Record<string, unknown>; events: StudioEvent[];
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
}) {
  const [items, setItems] = useState<TestItem[]>([]);
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState('');
  const running = project.runtime !== 'stopped';
  useEffect(() => {
    let cancelled = false;
    const refresh = () => api<{ items: TestItem[] }>('/project/test-panel')
      .then(result => { if (!cancelled) { setItems(result.items); setError(''); } })
      .catch(reason => { if (!cancelled) setError(reason.message); });
    refresh();
    const timer = running ? setInterval(refresh, 1000) : undefined;
    return () => { cancelled = true; clearInterval(timer); };
  }, [project.test_panel, running]);
  const save = (entries: TestControl[]) => perform(async () => {
    await prepare(); await api('/project/test-panel', 'PUT', entries); setAdding(false);
  });
  const entries = project.test_panel || [];
  return <main className="main-view"><div className="view-heading"><div><h1>Signal Test</h1>
    <p className="muted small">Your saved alarm, status and measurement controls. No drawing symbol required.</p></div>
    <button className="primary" disabled={running} onClick={() => setAdding(!adding)}>{adding ? 'Cancel adding' : 'Add signal control'}</button>
  </div><div className="content-page">
    <p className="small muted">These actions change simulated feedback directly; they are not MMS Open/Close commands. Reports follow the IED’s configured triggers.</p>
    {adding && !running && <section className="card test-picker"><SignalPicker ieds={project.ieds} choose={(binding, signal) => {
      if (!signal.writable || signal.fc === 'CF') { setError('Choose a runtime-editable signal; configuration and command attributes cannot be forced here.'); return; }
      save([...entries, { id: crypto.randomUUID(), label: (signal.description || signal.ref).slice(0, 100),
        ...binding, duration_ms: 1000, active_value: true }]);
    }} /></section>}
    {error && <div className="error-banner" role="alert">{error}</div>}
    {!items.length && <div className="empty"><h2>Build your test panel</h2><p>Add a signal, give it a clear label, then Run to set values or trigger a Boolean pulse.</p></div>}
    <div className="test-grid">{items.map((item, index) => <TestCard key={item.id} item={item} running={running}
      values={values} events={events} iedName={project.ieds.find(ied => ied.id === item.ied_id)?.name || 'Missing IED'}
      equipment={project.diagram.nodes.filter(node =>
        stateBindings(node.data.state_mapping).some(binding => binding.ied_id === item.ied_id && binding.ref === item.ref) ||
        relayBindings(node.data).some(binding => binding.ied_id === item.ied_id && binding.ref === item.ref))
        .map(node => node.data.label)}
      perform={perform} first={index === 0} last={index === items.length - 1}
      edit={entry => save(entries.map(current => current.id === entry.id ? entry : current))}
      remove={() => save(entries.filter(entry => entry.id !== item.id))}
      move={direction => { const next = [...entries]; [next[index], next[index + direction]] = [next[index + direction], next[index]]; save(next); }} />)}</div>
  </div></main>;
}
