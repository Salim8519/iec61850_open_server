import { useEffect, useState } from 'react';
import { Plus } from 'lucide-react';
import type { EquipmentData, Project, RelayStatus, Signal } from '../../api/types';
import { RelaySignalAdder } from './RelaySignalAdder';
import { RelayStatusRow } from './RelayStatusRow';
import { defaultRelayLabel } from '../../equipment/relayDisplay';
import { relayValueKey } from '../../equipment/stateBindings';
import { isMixedRelay, relayIedIds, resolveRelayIedId } from '../../equipment/relayIed';
import { api } from '../../api/client';
import './relay-signals.css';

const MAX_STATUSES = 50;

type Props = {
  nodeId: string; data: EquipmentData; project: Project; locked: boolean;
  values: Record<string, unknown>; update: (data: EquipmentData) => void;
  perform: (task: () => Promise<unknown>) => Promise<void>;
};

export function RelayStatusEditor({ nodeId, data, project, locked, values, update, perform }: Props) {
  const entries = data.relay_statuses || [];
  const [signals, setSignals] = useState<Record<string, Signal>>({});
  const [manual, setManual] = useState<Set<string>>(new Set());
  const [busyIds, setBusyIds] = useState<Set<string>>(new Set());
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [draftIed, setDraftIed] = useState('');
  const [switching, setSwitching] = useState(false);
  const [adding, setAdding] = useState(entries.length === 0);
  const storedIed = resolveRelayIedId(entries);
  const relayIedId = storedIed || draftIed;
  const relayIed = project.ieds.find(item => item.id === relayIedId);
  const mixed = isMixedRelay(entries);
  const signature = entries.map(entry => `${entry.source.ied_id}:${entry.source.ref}`).join('|');
  useEffect(() => {
    let cancelled = false;
    for (const key of signature.split('|').filter(Boolean)) {
      if (signals[key]) continue;
      const [iedId, ref] = key.split(/:(.+)/);
      api<{ items: Signal[] }>(`/ieds/${iedId}/signals?ref=${encodeURIComponent(ref)}`)
        .then(result => { if (!cancelled && result.items[0]) setSignals(current => ({ ...current, [key]: result.items[0] })); })
        .catch(reason => { if (!cancelled) setError(reason.message); });
    }
    return () => { cancelled = true; };
  }, [signature]);
  const setEntries = (list: RelayStatus[]) => update({ ...data, relay_statuses: list });
  const markBusy = (id: string, busy: boolean) => setBusyIds(current => {
    const next = new Set(current);
    if (busy) next.add(id); else next.delete(id);
    return next;
  });
  const initiate = (entry: RelayStatus, path: string, extra: Record<string, unknown> = {}) => {
    const key = `${entry.source.ied_id}:${entry.source.ref}`;
    setError(''); setNotice(''); markBusy(entry.id, true);
    perform(async () => {
      try {
        await api(path, 'POST', { ied_id: entry.source.ied_id, ref: entry.source.ref, ...extra });
        if (path !== '/signals/release-override') setManual(current => new Set(current).add(key));
        else setManual(current => { const next = new Set(current); next.delete(key); return next; });
      } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); throw reason; }
      finally { markBusy(entry.id, false); }
    });
  };
  const move = (index: number, direction: number) => {
    const next = [...entries];
    const target = index + direction;
    if (target < 0 || target >= next.length) return;
    [next[index], next[target]] = [next[target], next[index]];
    setEntries(next);
  };
  const switchIed = (nextId: string) => {
    setError(''); setNotice('');
    if (!entries.length) { setDraftIed(nextId); return; }
    const nextIed = project.ieds.find(item => item.id === nextId);
    if (!nextIed || nextId === storedIed) return;
    if (!window.confirm(`Reassign all ${entries.length} signals to ${nextIed.name}? Missing ones are dropped.`)) return;
    setSwitching(true);
    Promise.all(entries.map(entry =>
      api<{ items: Signal[] }>(`/ieds/${nextId}/signals?ref=${encodeURIComponent(entry.source.ref)}`)
        .then(result => ({ entry, found: !!result.items[0] }))
        .catch(() => ({ entry, found: false }))))
      .then(results => {
        const kept = results.filter(item => item.found).map(item => ({
          ...item.entry, source: { ied_id: nextId, report_ref: null, ref: item.entry.source.ref } }));
        const dropped = results.length - kept.length;
        setEntries(kept); setDraftIed('');
        setNotice(dropped ? `${dropped} signal${dropped > 1 ? 's' : ''} not found in ${nextIed.name}; pick replacements below.` : '');
      })
      .catch(reason => setError(reason instanceof Error ? reason.message : String(reason)))
      .finally(() => setSwitching(false));
  };
  const addSignal = (binding: RelayStatus['source'], signal: Signal) => {
    const id = crypto.randomUUID();
    setEntries([...entries, { id, label: defaultRelayLabel(signal, relayIed?.name), source: binding }]);
    setNotice('');
  };
  const card = (item: RelayStatus, index: number) => {
    const key = `${item.source.ied_id}:${item.source.ref}`;
    return <RelayStatusRow key={item.id} entry={item}
      iedName={project.ieds.find(ied => ied.id === item.source.ied_id)?.name} index={index} count={entries.length}
      locked={locked} signal={signals[key]} liveKey={relayValueKey(nodeId, item.id)} fallback={values[key]} busy={busyIds.has(item.id)}
      manual={manual.has(key)}
      rename={label => setEntries(entries.map(current =>
        current.id === item.id ? { ...current, label } : current))}
      move={direction => move(index, direction)}
      remove={() => setEntries(entries.filter(current => current.id !== item.id))}
      initiate={(path, extra) => initiate(item, path, extra)} />;
  };
  const canAdd = !locked && entries.length < MAX_STATUSES;
  return <div className="relay-editor">
    {!locked ? <label>Relay IED<select value={relayIedId} disabled={switching}
      onChange={e => switchIed(e.target.value)}>
      <option value="">Choose an IED</option>
      {project.ieds.map(ied => <option key={ied.id} value={ied.id}>{ied.name}</option>)}
    </select></label> :
      <p className="small muted relay-editor-ied">IED <strong>{relayIed?.name || relayIedIds(entries)[0] || '—'}</strong>
        <span>Set a value or pulse a triggerable signal. Display-only signals show live values.</span></p>}
    {mixed && !locked && <p className="badge warning" role="status">Mixed IEDs · choose one IED to reassign all rows</p>}
    {entries.length > 0 && <div className="relay-cards">{entries.map(card)}</div>}
    {notice && <p className="small" role="status">{notice}</p>}
    {error && <p className="relay-error" role="alert">{error}</p>}
    {!entries.length && <p className="small muted">{locked ? 'This relay has no signals.' :
      'No signals yet. Add the relay signals you want to watch or trigger.'}</p>}
    {canAdd && !relayIed && <p className="small muted">Choose an IED above to add signals.</p>}
    {canAdd && relayIed && (adding ? <div className="relay-adder">
      <div className="relay-adder-head"><strong>Add from {relayIed.name}</strong>
        <span className="small muted">{entries.length}/{MAX_STATUSES}</span>
        <button className="ghost" onClick={() => setAdding(false)}>Done</button></div>
      <RelaySignalAdder ied={relayIed}
        existing={new Set(entries.map(item => `${item.source.ied_id}:${item.source.ref}`))}
        disabled={switching} onAdd={addSignal} /></div> :
      <button className="relay-add" onClick={() => setAdding(true)}><Plus size={15} />Add signals</button>)}
    {!locked && entries.length >= MAX_STATUSES && <p className="small muted">Signal limit reached ({MAX_STATUSES}).</p>}
  </div>;
}
