import type { EquipmentData } from '../../api/types';
import { Disclosure } from '../../ui/Disclosure';
import { AppearanceEditor } from './AppearanceEditor';
import { Hint } from '../../ui/Hint';
import { advancedFields, basicFields, INFEED_TYPES, polaritySentence, PUSH_MODES } from './equipmentFields';

type Props = {
  data: EquipmentData; locked: boolean; update: (data: EquipmentData) => void;
  applyColors: (active: string | null, inactive: string | null) => void;
};

/** Name and start position up front; everything rare inside one Advanced disclosure. */
export function BasicSettings({ data, locked, update }: Omit<Props, 'applyColors'>) {
  const fields = basicFields(data.kind);
  return <>
    <label>Name<input value={data.label} disabled={locked} maxLength={100}
      onChange={event => update({ ...data, label: event.target.value })} /></label>
    {fields.includes('start_position') && <label>When the project starts<select disabled={locked}
      value={data.initial_position || 'open'}
      onChange={event => update({ ...data, initial_position: event.target.value as 'open' | 'closed' })}>
      <option value="open">Open</option><option value="closed">Closed</option></select></label>}
    {fields.includes('infeed_type') && <label>What feeds the station here<select disabled={locked}
      value={data.infeed_type || 'line'}
      onChange={event => update({ ...data, infeed_type: event.target.value as NonNullable<EquipmentData['infeed_type']> })}>
      {INFEED_TYPES.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>}
    {fields.includes('infeed_state') && <label>Power from here<select disabled={locked}
      value={data.infeed_state || 'live'}
      onChange={event => update({ ...data, infeed_state: event.target.value as 'live' | 'dead' })}>
      <option value="live">Live: it feeds power</option><option value="dead">Dead: no power</option></select></label>}
    {data.kind === 'INFEED' && <Hint id="infeed.intro" compact />}
    {data.kind === 'TRIP_RELAY' && <p className="small muted" style={{ margin: 0 }}>{polaritySentence(data)}</p>}
  </>;
}

export function AdvancedSettings({ data, locked, update, applyColors }: Props) {
  const fields = advancedFields(data);
  if (!fields.length) return null;
  const mode = data.push_mode || 'latching_status';
  const setMode = (next: typeof mode) => update({ ...data, push_mode: next,
    state_mapping: next === 'spc_momentary' ? null : data.state_mapping,
    control_binding: next === 'spc_momentary' ? data.control_binding : null });
  return <Disclosure>
    {fields.includes('push_mode') && <label>What the button does<select disabled={locked} value={mode}
      onChange={event => setMode(event.target.value as typeof mode)}>
      {PUSH_MODES.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}</select>
      <span className="small muted" style={{ fontWeight: 400 }}>Changing this clears a connection that no longer fits.</span></label>}
    {fields.includes('push_value') && <label>Press sends<select disabled={locked} value={String(data.push_value ?? true)}
      onChange={event => update({ ...data, push_value: event.target.value === 'true' })}>
      <option value="true">1 (true)</option><option value="false">0 (false)</option></select></label>}
    {fields.includes('push_rest_value') && <label>Rests at<select disabled={locked} value={String(data.push_rest_value ?? false)}
      onChange={event => update({ ...data, push_rest_value: event.target.value === 'true' })}>
      <option value="false">0 · released</option><option value="true">1 · pressed</option></select></label>}
    {fields.includes('push_reset_ms') && <label>Releases after (ms)<input type="number" min={100} max={60000} step={100}
      disabled={locked} value={data.push_reset_ms ?? 1000}
      onChange={event => update({ ...data, push_reset_ms: Number(event.target.value) })} /></label>}
    {fields.includes('trip_command_polarity') && <label>Command values<select disabled={locked}
      value={data.trip_command_polarity || 'operate_false'}
      onChange={event => update({ ...data, trip_command_polarity: event.target.value as 'operate_false' | 'operate_true' })}>
      <option value="operate_false">Operate sends 0 · Reset sends 1</option>
      <option value="operate_true">Operate sends 1 · Reset sends 0</option></select></label>}
    {fields.includes('trip_state_polarity') && <label>State values<select disabled={locked}
      value={data.trip_state_polarity || 'operated_true'}
      onChange={event => update({ ...data, trip_state_polarity: event.target.value as 'operated_false' | 'operated_true' })}>
      <option value="operated_true">Operated is 1 · Reset is 0</option>
      <option value="operated_false">Operated is 0 · Reset is 1</option></select></label>}
    {fields.includes('colors') && <div className="stack" style={{ gap: 8 }}><span className="section-label">Colours</span>
      <AppearanceEditor data={data} locked={locked} update={update} applyToKind={applyColors} /></div>}
  </Disclosure>;
}
