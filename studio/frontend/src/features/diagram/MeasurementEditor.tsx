import { useState } from 'react';
import { Plus } from 'lucide-react';
import type { EquipmentData, Measurement, Project, Signal, SignalBinding } from '../../api/types';
import { SignalSearchPicker } from '../../components/SignalSearchPicker';
import { applyToRows, MAX_MEASUREMENTS, MEASUREMENT_COPY, moveRow, newMeasurement, NUMERIC_KINDS }
  from '../../equipment/measurementDisplay';
import { MeasurementBulkBar } from './MeasurementBulkBar';
import { MeasurementRow } from './MeasurementRow';
import './relay-signals.css';
import './measurement-editor.css';

type Props = {
  nodeId: string; data: EquipmentData; project: Project; locked: boolean;
  update: (data: EquipmentData) => void;
};

/** Measurement list rows: add many numeric points at once, then scale, name and order them. */
export function MeasurementEditor({ nodeId, data, project, locked, update }: Props) {
  const entries = data.measurements || [];
  const [adding, setAdding] = useState(entries.length === 0);
  const [ticked, setTicked] = useState<ReadonlySet<string>>(new Set());
  const [notice, setNotice] = useState('');
  const selected = new Set([...ticked].filter(id => entries.some(entry => entry.id === id)));
  const iedName = (id: string) => project.ieds.find(ied => ied.id === id)?.name;
  const mixed = new Set(entries.map(entry => entry.source.ied_id)).size > 1;
  const setEntries = (list: Measurement[]) => update({ ...data, measurements: list });
  const add = (items: { binding: SignalBinding; signal: Signal }[]) => {
    const room = MAX_MEASUREMENTS - entries.length;
    const fresh = items.slice(0, room).map(item => newMeasurement(item.binding, item.signal, iedName(item.binding.ied_id)));
    setEntries([...entries, ...fresh]);
    setNotice(items.length > room ? `Only ${room} more fit: a list holds at most ${MAX_MEASUREMENTS} measurements.` :
      items.length > 1 ? `${fresh.length} measurements added.` : '');
  };
  const toggle = (id: string) => setTicked(current => {
    const next = new Set(current);
    if (next.has(id)) next.delete(id); else next.add(id);
    return next;
  });
  const allTicked = entries.length > 0 && selected.size === entries.length;
  const canAdd = !locked && entries.length < MAX_MEASUREMENTS;
  return <div className="relay-editor measure-editor">
    {!locked && <p className="small muted measure-hint">{MEASUREMENT_COPY.scaleHint}</p>}
    {!locked && entries.length > 1 && <label className="measure-tick-all">
      <input type="checkbox" checked={allTicked} aria-label="Tick all measurements" onChange={() =>
        setTicked(allTicked ? new Set() : new Set(entries.map(entry => entry.id)))} />
      <span>{selected.size ? allTicked ? 'All ticked' : 'Tick all' : `Tick all · ${MEASUREMENT_COPY.bulkHint}`}</span></label>}
    {!locked && selected.size > 0 && <MeasurementBulkBar entries={entries} selected={selected}
      apply={patch => setEntries(applyToRows(entries, selected, patch))} clear={() => setTicked(new Set())} />}
    {entries.length > 0 && <div className="relay-cards">{entries.map((entry, index) =>
      <MeasurementRow key={entry.id} nodeId={nodeId} entry={entry} iedName={iedName(entry.source.ied_id)}
        showIed={mixed} index={index} count={entries.length} locked={locked} selected={selected.has(entry.id)}
        toggle={() => toggle(entry.id)}
        change={patch => setEntries(entries.map(item => item.id === entry.id ? { ...item, ...patch } : item))}
        move={direction => setEntries(moveRow(entries, index, direction))}
        remove={() => setEntries(entries.filter(item => item.id !== entry.id))} />)}</div>}
    {notice && !locked && <p className="small" role="status">{notice}</p>}
    {!entries.length && <p className="small muted">{locked ? 'This list has no measurements.' :
      'No measurements yet. Choose an IED below and add the values you want to watch.'}</p>}
    {canAdd && (adding ? <div className="relay-adder">
      <div className="relay-adder-head"><strong>Add number signals</strong>
        <span className="small muted">{entries.length}/{MAX_MEASUREMENTS}</span>
        <button className="ghost" onClick={() => setAdding(false)}>Done</button></div>
      <SignalSearchPicker ieds={project.ieds} allowedKinds={NUMERIC_KINDS} chooseLabel="Add"
        memoryKey={`sld-measurements-${nodeId}`}
        exclude={new Set(entries.map(entry => `${entry.source.ied_id}:${entry.source.ref}`))}
        choose={(binding, signal) => add([{ binding, signal }])} chooseAll={add} /></div> :
      <button className="relay-add" onClick={() => setAdding(true)}><Plus size={15} />Add measurements</button>)}
    {!locked && entries.length >= MAX_MEASUREMENTS && <p className="small muted">
      Measurement limit reached ({MAX_MEASUREMENTS}). Add another measurement list for more.</p>}
  </div>;
}
