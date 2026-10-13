import type { PointerEvent as ReactPointerEvent } from 'react';
import { MousePointer2, Link2, Copy, Trash2 } from 'lucide-react';
import type { EquipmentNode, Project } from '../../api/types';
import { kindInfo } from '../../equipment/kindInfo';
import { InspectorHeader } from './InspectorHeader';
import { AppearanceEditor } from './AppearanceEditor';
import { defaultScheme } from '../../equipment/stateColors';
import './inspector.css';
import { StateMappingEditor } from './StateMappingEditor';
import { LEGACY_KINDS } from '../../equipment/registry';
import { ControlMapping } from './ControlMapping';
import { MeasurementSourceEditor } from './MeasurementSourceEditor';
import { RelayStatusEditor } from './RelayStatusEditor';
import { PropertySection } from './PropertySection';

type Props = {
  node?: EquipmentNode; selectionCount: number; project: Project; locked: boolean; busy: boolean;
  values: Record<string, unknown>; operation?: string;
  panelWidth: number | null; setPanelWidth: (width: number | null) => void;
  update: (data: EquipmentNode['data']) => void; remove: () => void; duplicate: () => void;
  operate: (nodeId: string, action: string) => void;
  applyColors: (active: string | null, inactive: string | null) => void;
  perform: (task: () => Promise<unknown>) => Promise<void>;
};

const MIN_WIDTH = 280;
const MAX_WIDTH = 640;
const clamp = (width: number) => Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, Math.round(width)));

function ResizeHandle({ panelWidth, setPanelWidth }: {
  panelWidth: number | null; setPanelWidth: (width: number | null) => void;
}) {
  const drag = (event: ReactPointerEvent<HTMLDivElement>) => {
    const panel = (event.target as HTMLElement).closest('.inspector-panel') as HTMLElement | null;
    const startX = event.clientX;
    const startWidth = panel?.offsetWidth || panelWidth || 320;
    (event.target as HTMLElement).setPointerCapture(event.pointerId);
    const move = (next: PointerEvent) => setPanelWidth(clamp(startWidth + (startX - next.clientX)));
    const stop = () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', stop);
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', stop);
  };
  return <div className="inspector-resize" role="separator" aria-orientation="vertical"
    aria-label="Resize properties panel" title="Drag to resize · double-click to reset"
    tabIndex={0} onPointerDown={drag} onDoubleClick={() => setPanelWidth(null)}
    onKeyDown={event => {
      if (event.key === 'ArrowLeft') setPanelWidth(clamp((panelWidth || 320) + 16));
      if (event.key === 'ArrowRight') setPanelWidth(clamp((panelWidth || 320) - 16));
      if (event.key === 'Home') setPanelWidth(null);
    }} />;
}

export function Inspector({ node, selectionCount, project, locked, busy, values, operation,
  panelWidth, setPanelWidth, update, remove, duplicate, operate, perform, applyColors }: Props) {
  const style = panelWidth ? { width: panelWidth } : undefined;
  if (!node) return <div className="inspector-panel" style={style}><ResizeHandle
    panelWidth={panelWidth} setPanelWidth={setPanelWidth} /><aside className="inspector">
    <span className="section-label">Properties</span>
    {selectionCount > 1 ? <div className="empty"><MousePointer2 size={28} /><h2>{selectionCount} items selected</h2>
      <p>Drag to move together. Ctrl+C / Ctrl+V to copy, Ctrl+D to duplicate, arrows to nudge.</p>
      <div className="row" style={{ justifyContent: 'center' }}>
        <button disabled={locked} onClick={duplicate}><Copy size={14} />Duplicate</button>
        <button disabled={locked} onClick={remove}><Trash2 size={14} />Delete</button></div></div> :
    <div className="empty"><MousePointer2 size={28} /><h2>Select equipment</h2>
      <p>Edit its properties and link it to an IED signal. Drag on empty canvas to select several.</p></div>}
    <div className="inspector-note"><Link2 size={16} /><p>Equipment and IEDs are independent. Connect them with signal mappings.</p></div>
  </aside></div>;
  const data = node.data;
  const junction = data.kind === 'JUNCTION';
  const relay = data.kind === 'RELAY';
  const legacy = (LEGACY_KINDS as string[]).includes(data.kind);
  const switching = ['CB', 'DS', 'ES'].includes(data.kind);
  const trip = data.kind === 'TRIP_RELAY';
  const push = data.kind === 'PUSH_BUTTON';
  const pushMode = data.push_mode || 'latching_status';
  const latching = push && pushMode === 'latching_status';
  const selfReset = push && pushMode === 'self_resetting';
  const statusButton = push && pushMode !== 'spc_momentary';
  const showState = !junction && !relay && (!push || statusButton);
  const showControl = switching || trip || (push && !statusButton);
  const canAct = statusButton ? !!data.state_mapping : !!data.control_binding;
  const setPushMode = (mode: 'spc_momentary' | 'latching_status' | 'self_resetting') => update({ ...data,
    push_mode: mode, state_mapping: mode === 'spc_momentary' ? null : data.state_mapping,
    control_binding: mode === 'spc_momentary' ? data.control_binding : null,
  });
  const info = kindInfo(data.kind);
  const stateHint = data.kind === 'SOURCE' ? 'Numeric MX point this source drives.' :
    switching ? 'Dbpos point, one Boolean, or separate open/closed contacts.' :
    trip || push ? 'Boolean status point shown on the canvas.' : 'IED value shown on the canvas.';
  const liveSection = locked && canAct && <PropertySection tone="live"
      title={latching ? 'Live status' : 'Operate through MMS'}>
      <div className="row">{switching ? <>
        <button disabled={busy} onClick={() => operate(node.id, 'open')}>Open</button>
        <button className="primary" disabled={busy} onClick={() => operate(node.id, 'close')}>Close</button>
      </> : trip ? <>
        <button disabled={busy} onClick={() => operate(node.id, 'reset')}>Reset</button>
        <button className="danger" disabled={busy} onClick={() => operate(node.id, 'operate')}>Operate</button>
      </> : <button className="primary" disabled={busy}
        onClick={() => operate(node.id, 'press')}>{latching ? 'Toggle' : 'Press'}</button>}</div>
      {operation && <p className="small" role="status">{operation}</p>}
    </PropertySection>;
  return <div className="inspector-panel" style={style}><ResizeHandle
    panelWidth={panelWidth} setPanelWidth={setPanelWidth} />
    <aside className="inspector">
    <div className="section-label">Equipment properties</div>
    <InspectorHeader data={data} locked={locked} rotatable={!junction && !relay}
      rotate={() => update({ ...data, rotation: (data.rotation + 90) % 360 })}
      duplicate={duplicate} remove={remove} />
    {liveSection}
    {locked && <p className="inspector-locked" role="status">Running: structure and mappings are locked. Stop to edit.</p>}
    {legacy && <PropertySection tone="legacy" title="Legacy object">
      <p className="prop-hint">Kept for saved diagrams. Replace with a Junction or delete it.</p>
    </PropertySection>}
    <PropertySection tone="neutral" title="General">
      <label>Name<input value={data.label} disabled={locked}
        onChange={event => update({ ...data, label: event.target.value })} /></label>
      {trip && <><label>Trip command polarity<select disabled={locked}
        value={data.trip_command_polarity || 'operate_false'}
        onChange={event => update({ ...data,
          trip_command_polarity: event.target.value as 'operate_false' | 'operate_true' })}>
        <option value="operate_false">Operate sends 0 · Reset sends 1</option>
        <option value="operate_true">Operate sends 1 · Reset sends 0</option>
      </select></label><label>Trip state polarity<select disabled={locked}
        value={data.trip_state_polarity || 'operated_true'}
        onChange={event => update({ ...data,
          trip_state_polarity: event.target.value as 'operated_false' | 'operated_true' })}>
        <option value="operated_true">Operated is 1 / Reset is 0</option>
        <option value="operated_false">Operated is 0 / Reset is 1</option>
      </select></label></>}
      {push && <><label>Button behavior<select disabled={locked}
        value={pushMode}
        onChange={event => setPushMode(event.target.value as 'spc_momentary' | 'latching_status' | 'self_resetting')}>
        <option value="latching_status">Latching Boolean status</option>
        <option value="self_resetting">Self-resetting Boolean status</option>
        <option value="spc_momentary">Momentary SPC command</option>
      </select></label><p className="prop-hint">Changing behavior clears any incompatible mapping.</p>
      {pushMode === 'spc_momentary' && <label>Press sends<select disabled={locked}
        value={String(data.push_value ?? true)}
        onChange={event => update({ ...data, push_value: event.target.value === 'true' })}>
        <option value="true">1 (true)</option><option value="false">0 (false)</option>
      </select></label>}
      {selfReset && <><label>Rests at<select disabled={locked}
        value={String(data.push_rest_value ?? false)}
        onChange={event => update({ ...data, push_rest_value: event.target.value === 'true' })}>
        <option value="false">0 · released</option><option value="true">1 · pressed</option>
      </select></label><label>Reset after (ms)<input type="number" min={100} max={60000} step={100}
        disabled={locked} value={data.push_reset_ms ?? 1000}
        onChange={event => update({ ...data, push_reset_ms: Number(event.target.value) })} /></label></>}</>}
      {switching && <label>Start position<select disabled={locked}
        value={data.initial_position || 'open'}
        onChange={event => update({ ...data,
          initial_position: event.target.value as 'open' | 'closed' })}>
        <option value="open">Open</option><option value="closed">Closed</option>
      </select></label>}
    </PropertySection>
    {defaultScheme(data) && <PropertySection tone="neutral" title="Appearance"
      hint="Colour of the switching part for each confirmed state. The label still names the state.">
      <AppearanceEditor data={data} locked={locked} update={update} applyToKind={applyColors} /></PropertySection>}
    {showState && <PropertySection tone="ied" title={info.state} hint={stateHint}><StateMappingEditor
      key={`${node.id}:state:${data.push_mode}`}
      data={data} ieds={project.ieds} disabled={locked} update={update} /></PropertySection>}
    {relay && <PropertySection tone="ied" title="Relay signals"
      hint={locked ? undefined : 'Pick one IED, then add the signals to watch or trigger.'}><RelayStatusEditor key={node.id}
      nodeId={node.id} data={data} project={project} locked={locked}
      values={values} update={update} perform={perform} /></PropertySection>}
    {data.kind === 'SOURCE' && <PropertySection tone="ied" title="Value generation" badge="Simulation"
      hint="How the simulator drives this point."><MeasurementSourceEditor
      data={data} project={project} values={values} perform={perform} /></PropertySection>}
    {showControl && <PropertySection tone="command" title="Command"
      hint={`${trip || push ? 'SPC' : 'DPC'} control operated through MMS.`}><ControlMapping
      key={`${node.id}:control:${data.push_mode}`}
      data={data} ieds={project.ieds} disabled={locked} update={update} /></PropertySection>}
  </aside></div>;
}
