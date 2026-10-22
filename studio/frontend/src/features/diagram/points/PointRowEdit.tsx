import { useState } from 'react';
import { Trash2 } from 'lucide-react';
import type { PanelPoint } from '../../../api/types';
import { displayToRaw, pointSettingFromText, rawToInput } from '../../../equipment/pointDisplay';
import { DOUBLE_POSITIONS, MAX_POINT_GROUP, type PointPatch } from '../../../equipment/pointPanel';
import type { SettingKey } from '../../../equipment/measurementDisplay';
import { POINT_COPY } from '../../../copy/pointPanel';
import { DraftField } from '../DraftField';

const VALUE_FIELDS: { key: SettingKey; label: string; mode: 'decimal' | 'numeric' | 'text'; title: string }[] = [
  { key: 'scale', label: 'Scale ×', mode: 'decimal', title: 'The raw value is multiplied by this, e.g. 0.001 for V to kV' },
  { key: 'offset', label: 'Offset +', mode: 'decimal', title: 'Added after scaling, e.g. -4 to remove a 4 mA zero' },
  { key: 'unit', label: 'Unit', mode: 'text', title: 'Shown after the value, e.g. kV, A, MW, Hz' },
  { key: 'decimals', label: 'Decimals', mode: 'numeric', title: 'Digits after the point, 0 to 6' },
];

/** The healthy value of a status or double point is a choice; a value point types it. */
function HealthyField({ point, change }: { point: PanelPoint; change: (patch: PointPatch) => void }) {
  if (point.type === 'value') {
    return <DraftField label={`${POINT_COPY.healthyValue} (${point.unit || 'raw'})`} inputMode="decimal"
      value={point.healthy === null ? '' : rawToInput(point, point.healthy)} placeholder={POINT_COPY.healthyNone}
      title="The value All healthy writes, in the unit shown. Leave empty for none."
      commit={text => {
        if (!text.trim()) { change({ healthy: null }); return ''; }
        const result = displayToRaw(point, text);
        if (result.value === undefined) return result.error || 'Type a number.';
        change({ healthy: result.value }); return '';
      }} />;
  }
  const options = point.type === 'status' ? [['on', 'ON', true], ['off', 'OFF', false]] as const
    : DOUBLE_POSITIONS.map(item => [String(item.raw), item.word, item.raw] as const);
  const current = point.healthy === null ? '' : options.find(item => item[2] === point.healthy)?.[0] ?? '';
  return <label className="draft-field" title="The value All healthy writes. Choose None to leave this point alone.">
    <span>{POINT_COPY.healthyValue}</span>
    <select value={current} onChange={event => change({
      healthy: event.target.value === '' ? null : options.find(item => item[0] === event.target.value)?.[2] ?? null })}>
      <option value="">{POINT_COPY.healthyNone}</option>
      {options.map(item => <option key={item[0]} value={item[0]}>{item[1]}</option>)}
    </select></label>;
}

/** Edit fields of one row while the IEDs are stopped: group, healthy value, value settings, pin, remove. */
export function PointRowEdit({ point, change, remove }: {
  point: PanelPoint; change: (patch: PointPatch) => void; remove: () => void;
}) {
  const [problems, setProblems] = useState<Partial<Record<string, string>>>({});
  const problem = Object.values(problems).find(Boolean);
  const report = (key: string) => (error: string) => setProblems(current => ({ ...current, [key]: error }));
  return <div className="pp-edit">
    <div className="pp-edit-fields">
      <DraftField label={POINT_COPY.group} value={point.group} maxLength={MAX_POINT_GROUP} placeholder={POINT_COPY.noGroup}
        title="Points with the same group are listed together" report={report('group')}
        commit={text => { change({ group: text.trim() }); return ''; }} />
      {point.type !== 'command' && <HealthyField point={point} change={change} />}
      {point.type === 'value' && VALUE_FIELDS.map(field => <DraftField key={field.key} label={field.label}
        value={String(point[field.key])} inputMode={field.mode} title={field.title} maxLength={field.key === 'unit' ? 16 : 24}
        placeholder={field.key === 'unit' ? 'none' : undefined} report={report(field.key)}
        commit={text => {
          const result = pointSettingFromText(field.key, text);
          if (result.patch) change(result.patch);
          return result.error || '';
        }} />)}
    </div>
    {problem && <p className="relay-error" role="alert">{problem}</p>}
    <div className="pp-edit-tools">
      {point.type === 'value' && <label className="pp-pin" title={POINT_COPY.pinHint}>
        <input type="checkbox" checked={point.pinned} onChange={event => change({ pinned: event.target.checked })} />
        <span>{POINT_COPY.pin}</span></label>}
      <button className="icon ghost relay-remove" title={POINT_COPY.remove} aria-label={`${POINT_COPY.remove}: ${point.label}`}
        onClick={remove}><Trash2 size={15} /></button>
    </div>
  </div>;
}
