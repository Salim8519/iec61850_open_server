import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { Ied, Scenario, ScenarioStep, Signal } from '../../api/types';
import { runWords, stepAction, stepCount } from './sequenceWords';

export function SequenceEditor({ ieds, scenarios, running, perform }: {
  ieds: Ied[]; scenarios: Scenario[]; running: boolean;
  perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const fresh = (): Scenario => ({ id: crypto.randomUUID(), name: 'New sequence', enabled: false, steps: [] });
  const [draft, setDraft] = useState<Scenario>(fresh);
  const [iedId, setIedId] = useState(ieds[0]?.id || '');
  const [query, setQuery] = useState('');
  const [signals, setSignals] = useState<Signal[]>([]);
  const [ref, setRef] = useState('');
  const [at, setAt] = useState(0);
  const [action, setAction] = useState<'set' | 'assert'>('set');
  const [value, setValue] = useState('0');
  const [error, setError] = useState('');
  const [status, setStatus] = useState<{ id: string; state: string; completed_steps: number }[]>([]);
  useEffect(() => {
    let active = true;
    const timer = setTimeout(() => {
      if (!iedId) { setSignals([]); return; }
      api<{ items: Signal[] }>(`/ieds/${iedId}/signals?q=${encodeURIComponent(query)}&limit=100`)
        .then(result => { if (active) setSignals(result.items.filter(signal =>
          signal.kind.startsWith('FLOAT') || signal.kind.startsWith('INT') ||
          ['BOOLEAN', 'Dbpos', 'Enum', 'Quality', 'Timestamp'].includes(signal.kind))); })
        .catch(reason => { if (active) setError(reason.message); });
    }, 200);
    return () => { active = false; clearTimeout(timer); };
  }, [iedId, query]);
  useEffect(() => {
    let active = true;
    const refresh = () => api<typeof status>('/scenarios/runs')
      .then(result => { if (active) setStatus(result); }).catch(reason => { if (active) setError(reason.message); });
    refresh();
    const timer = setInterval(refresh, 1000);
    return () => { active = false; clearInterval(timer); };
  }, []);
  const addStep = () => {
    const signal = signals.find(item => item.ref === ref);
    if (!signal) { setError('Choose a signal from the search results.'); return; }
    if (signal.kind === 'BOOLEAN' && !['true', 'false'].includes(value)) {
      setError('Boolean values must be true or false.'); return;
    }
    if (signal.kind !== 'BOOLEAN' && (!value.trim() || !Number.isFinite(Number(value)))) {
      setError('Enter a finite numeric value.'); return;
    }
    const step: ScenarioStep = { id: crypto.randomUUID(), at, ied_id: iedId, ref, action,
      value: signal.kind === 'BOOLEAN' ? value === 'true' : Number(value) };
    setDraft({ ...draft, steps: [...draft.steps, step].sort((left, right) => left.at - right.at) }); setError('');
  };
  return <section className="card">
    <div className="card-heading"><h2>Timed sequences</h2></div>
    <div className="stack" style={{ padding: 20 }}>
      <p className="muted" style={{ margin: 0 }}>A sequence sets or checks values at set times after you press Run.
        A failed check stops the sequence.</p>
      {scenarios.map(scenario => <div className="row" key={scenario.id}>
        <strong>{scenario.name}</strong><span className="muted small">{stepCount(scenario.steps.length)} · {(() => {
          const run = status.find(item => item.id === scenario.id);
          return runWords(run?.state, run?.completed_steps, scenario.steps.length);
        })()}</span>
        <button disabled={running} onClick={() => { setDraft(structuredClone(scenario)); setError(''); }}>Edit</button>
        <button disabled={!running} onClick={() => perform(() => api(`/scenarios/${scenario.id}/start`, 'POST'))}>Run sequence</button>
        <button disabled={running} onClick={() => perform(() => api('/scenarios', 'PUT', scenarios.filter(item => item.id !== scenario.id)))}>Delete</button>
      </div>)}
      <fieldset disabled={running} className="stack" style={{ border: 0, padding: 0, margin: 0 }}>
        <div className="row"><label style={{ flex: 1 }}>Sequence name<input value={draft.name}
          onChange={event => setDraft({ ...draft, name: event.target.value })} /></label>
          <button onClick={() => setDraft(fresh())}>New sequence</button></div>
        <label className="row"><input type="checkbox" style={{ width: 18, minHeight: 18 }} checked={draft.enabled}
          onChange={event => setDraft({ ...draft, enabled: event.target.checked })} />Start automatically with Run</label>
        <div className="generator-grid">
          <label>Time (seconds)<input type="number" min="0" max="86400" step="0.1" value={at} onChange={e => setAt(Number(e.target.value))} /></label>
          <label>Action<select value={action} onChange={e => setAction(e.target.value as 'set' | 'assert')}>
            <option value="set">Set value</option><option value="assert">Check value</option></select></label>
          <label>IED<select value={iedId} onChange={e => { setIedId(e.target.value); setRef(''); }}>
            <option value="">Choose IED</option>{ieds.map(ied => <option value={ied.id} key={ied.id}>{ied.name}</option>)}</select></label>
          <label>Search references<input value={query} onChange={e => { setQuery(e.target.value); setRef(''); }} /></label>
        </div>
        <div className="row"><label style={{ flex: 1 }}>Signal<select value={ref} onChange={e => {
          setRef(e.target.value); setValue(String(signals.find(item => item.ref === e.target.value)?.value ?? 0));
        }}><option value="">Choose signal</option>{signals.map(signal => <option key={signal.ref} value={signal.ref}>{signal.ref} [{signal.fc}]</option>)}</select></label>
          <label>Value<input value={value} onChange={e => setValue(e.target.value)} /></label><button disabled={!ref} onClick={addStep}>Add step</button></div>
        <div className="table-scroll"><table><thead><tr><th>Seconds</th><th>Action</th><th>Reference</th><th>Value</th><th /></tr></thead>
          <tbody>{draft.steps.map(step => <tr key={step.id}><td>{step.at}</td><td>{stepAction(step.action)}</td><td><code>{step.ref}</code></td>
            <td>{String(step.value)}</td><td><button onClick={() => setDraft({ ...draft, steps: draft.steps.filter(item => item.id !== step.id) })}>Remove</button></td></tr>)}</tbody></table></div>
        {error && <p role="alert" style={{ color: 'var(--error)' }}>{error}</p>}
        <button className="primary" disabled={!draft.steps.length} onClick={() => perform(() => api('/scenarios', 'PUT',
          [...scenarios.filter(item => item.id !== draft.id), draft]))}>Save sequence</button>
      </fieldset>
      {running && <span className="muted small">Stop the simulation to edit sequences.</span>}
    </div>
  </section>;
}
