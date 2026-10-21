import { useState } from 'react';
import { ArrowDown, ArrowUp, Trash2 } from 'lucide-react';
import type { Measurement } from '../../api/types';
import { MeasurementValue, measurementName, useMeasurementLive } from '../../equipment/MeasurementPanel';
import { settingFromText, type MeasurementPatch, type SettingKey } from '../../equipment/measurementDisplay';
import { shortRef } from '../../equipment/relayDisplay';
import { useSignalValue } from '../../state/useLive';
import { DraftField } from './DraftField';

type Props = {
  nodeId: string; entry: Measurement; iedName?: string; showIed: boolean; index: number; count: number;
  locked: boolean; selected: boolean; toggle: () => void;
  change: (patch: Partial<Measurement>) => void; move: (direction: number) => void; remove: () => void;
};

const FIELDS: { key: SettingKey; label: string; mode: 'decimal' | 'numeric' | 'text'; title: string }[] = [
  { key: 'scale', label: 'Scale ×', mode: 'decimal', title: 'The raw value is multiplied by this, e.g. 0.001 for V to kV' },
  { key: 'offset', label: 'Offset +', mode: 'decimal', title: 'Added after scaling, e.g. -4 to remove a 4 mA zero' },
  { key: 'unit', label: 'Unit', mode: 'text', title: 'Shown after the value, e.g. kV, A, MW, Hz' },
  { key: 'decimals', label: 'Decimals', mode: 'numeric', title: 'Digits after the point, 0 to 6' },
];

/** One measurement: tick for bulk edit, name, value now, scaling fields while stopped. */
export function MeasurementRow({ nodeId, entry, iedName, showIed, index, count, locked, selected, toggle,
  change, move, remove }: Props) {
  const live = useMeasurementLive(nodeId, entry);
  const fallback = useSignalValue(`${entry.source.ied_id}:${entry.source.ref}`);
  const raw = live.raw === undefined ? fallback : live.raw;
  const [problems, setProblems] = useState<Partial<Record<SettingKey, string>>>({});
  const commit = (key: SettingKey) => (text: string) => {
    const result = settingFromText(key, text);
    if (result.patch) change(result.patch as MeasurementPatch);
    return result.error || '';
  };
  const problem = Object.values(problems).find(Boolean);
  const name = measurementName(entry, iedName);
  return <div className={`measure-row${selected ? ' selected' : ''}`}>
    <div className="measure-row-head">
      {locked ? <span className="relay-card-index" aria-hidden="true">{index + 1}</span> :
        <input type="checkbox" checked={selected} onChange={toggle} aria-label={`Select ${name}`}
          title="Tick to change several measurements together" />}
      {locked ? <strong className="measure-row-name">{name}</strong> :
        <input className="measure-row-name" maxLength={100} value={name} aria-label={`Measurement ${index + 1} name`}
          placeholder="Name this measurement" title="Rename"
          onChange={event => change({ label: event.target.value || entry.source.ref })} />}
      {locked && <MeasurementValue raw={raw} entry={entry} stopped={live.stopped} bad={live.bad} />}
    </div>
    {locked ? <p className="measure-row-summary">
      × {entry.scale} {entry.offset < 0 ? '−' : '+'} {Math.abs(entry.offset)} · {entry.unit || 'no unit'} · {entry.decimals} decimals
      {live.bad && <strong> · Bad quality</strong>}</p> :
      <div className="measure-row-fields">{FIELDS.map(field => <DraftField key={field.key} label={field.label}
        value={String(entry[field.key])} inputMode={field.mode} title={field.title}
        maxLength={field.key === 'unit' ? 16 : 24} placeholder={field.key === 'unit' ? 'none' : undefined}
        commit={commit(field.key)}
        report={error => setProblems(current => ({ ...current, [field.key]: error }))} />)}</div>}
    {problem && <p className="relay-error" role="alert">{problem}</p>}
    <div className="measure-row-foot">
      {!locked && <span className="measure-row-now">Now <MeasurementValue raw={raw} entry={entry}
        stopped={live.stopped} bad={live.bad} /></span>}
      <p className="measure-row-meta" title={entry.source.ref}>
        {showIed && iedName && <span>{iedName} · </span>}<code>{shortRef(entry.source.ref, iedName)}</code>
        {live.signal && <span> · {live.signal.kind} · {live.signal.fc}</span>}</p>
      {!locked && <span className="relay-card-tools">
        <button className="icon ghost" title="Move up" aria-label={`Move ${name} up`}
          disabled={index === 0} onClick={() => move(-1)}><ArrowUp size={14} /></button>
        <button className="icon ghost" title="Move down" aria-label={`Move ${name} down`}
          disabled={index === count - 1} onClick={() => move(1)}><ArrowDown size={14} /></button>
        <button className="icon ghost relay-remove" title="Remove measurement" aria-label={`Remove ${name}`}
          onClick={remove}><Trash2 size={14} /></button></span>}
    </div>
  </div>;
}
