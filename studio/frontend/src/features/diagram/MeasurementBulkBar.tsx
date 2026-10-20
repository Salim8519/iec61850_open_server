import { useState } from 'react';
import type { Measurement } from '../../api/types';
import { SCALE_PRESETS, settingFromText, sharedSettings, type MeasurementPatch, type SettingKey }
  from '../../equipment/measurementDisplay';
import { DraftField } from './DraftField';

type Props = {
  entries: Measurement[]; selected: ReadonlySet<string>;
  apply: (patch: MeasurementPatch) => void; clear: () => void;
};

const FIELDS: { key: SettingKey; label: string; mode: 'decimal' | 'numeric' | 'text' }[] = [
  { key: 'scale', label: 'Scale ×', mode: 'decimal' },
  { key: 'offset', label: 'Offset +', mode: 'decimal' },
  { key: 'unit', label: 'Unit', mode: 'text' },
  { key: 'decimals', label: 'Decimals', mode: 'numeric' },
];

/** Same scale, offset, unit or decimals for every ticked row; a field left blank changes nothing. */
export function MeasurementBulkBar({ entries, selected, apply, clear }: Props) {
  const shared = sharedSettings(entries, selected);
  const [problem, setProblem] = useState('');
  const commit = (key: SettingKey) => (text: string) => {
    if (text.trim() === '' && key !== 'unit') return '';
    const result = settingFromText(key, text);
    if (result.patch) apply(result.patch);
    return result.error || '';
  };
  return <div className="measure-bulk" role="group" aria-label="Change the ticked measurements">
    <div className="measure-bulk-head">
      <strong>{selected.size} ticked</strong>
      <span className="small muted">Changes apply to all of them at once.</span>
      <button className="ghost" onClick={clear}>Untick all</button>
    </div>
    <div className="measure-bulk-presets">
      <span className="small muted">Quick scale</span>
      {SCALE_PRESETS.map(preset => <button key={preset.scale} title={preset.hint}
        className={shared.scale === preset.scale ? 'active' : ''} aria-pressed={shared.scale === preset.scale}
        onClick={() => apply({ scale: preset.scale })}>{preset.label}</button>)}
    </div>
    <div className="measure-row-fields">{FIELDS.map(field => <DraftField key={field.key} label={field.label}
      value={shared[field.key] === undefined ? '' : String(shared[field.key])} inputMode={field.mode}
      placeholder={shared[field.key] === undefined ? 'mixed' : undefined}
      title={shared[field.key] === undefined ? 'The ticked rows differ here; type a value to give them all the same' : undefined}
      maxLength={field.key === 'unit' ? 16 : 24} commit={commit(field.key)} report={setProblem} />)}</div>
    {problem && <p className="relay-error" role="alert">{problem}</p>}
  </div>;
}
