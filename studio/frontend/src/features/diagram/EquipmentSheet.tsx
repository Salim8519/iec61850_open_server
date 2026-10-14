import { Copy, RotateCw, Share2, Trash2, X } from 'lucide-react';
import type { EquipmentNode, Project } from '../../api/types';
import { kindInfo, setupItems } from '../../equipment/kindInfo';
import { EquipmentSymbol } from '../../equipment/symbols/EquipmentSymbol';
import { StatusPill } from '../../ui/StatusPill';
import { needsStatus, UI_STATUS } from '../../copy/status';
import { SourceEditor } from '../test/SourceEditor';
import { ControlMapping } from './ControlMapping';
import { RelayStatusEditor } from './RelayStatusEditor';
import { AdvancedSettings, BasicSettings } from './SheetSettings';
import { StateMappingEditor } from './StateMappingEditor';

type Props = {
  node: EquipmentNode; project: Project; locked: boolean; busy: boolean; values: Record<string, unknown>;
  operation?: string; update: (data: EquipmentNode['data']) => void; remove: () => void; duplicate: () => void;
  operate: (nodeId: string, action: string) => void; close: () => void;
  applyColors: (active: string | null, inactive: string | null) => void;
  perform: (task: () => Promise<unknown>) => Promise<void>;
  showFlow?: (iedId: string) => void;
};

export function EquipmentSheet({ node, project, locked, busy, values, operation, update, remove, duplicate,
  operate, close, applyColors, perform, showFlow }: Props) {
  const data = node.data;
  const info = kindInfo(data.kind);
  const switching = ['CB', 'DS', 'ES'].includes(data.kind);
  const trip = data.kind === 'TRIP_RELAY';
  const push = data.kind === 'PUSH_BUTTON';
  const statusButton = push && data.push_mode !== 'spc_momentary';
  const showState = !['JUNCTION', 'RELAY'].includes(data.kind) && (!push || statusButton);
  const showControl = switching || trip || (push && !statusButton);
  const canAct = statusButton ? !!data.state_mapping : !!data.control_binding;
  const missing = setupItems(data).filter(item => !item.done);
  const linkedIed = data.control_binding?.ied_id ?? (data.state_mapping?.mode === 'contacts_to_dbpos'
    ? data.state_mapping.open.ied_id : data.state_mapping?.source.ied_id);
  const source = data.kind === 'SOURCE' && data.state_mapping?.mode === 'direct' ? data.state_mapping.source : null;
  return <aside className={`sheet cat-${info.category}`} aria-label={`${data.label} properties`}>
    <div className="sheet-head">
      <span className="sheet-head-symbol"><EquipmentSymbol kind={data.kind} size={34} /></span>
      <div className="sheet-title"><h2 title={data.label}>{data.label}</h2><p>{info.name}</p></div>
      <button className="icon ghost" aria-label="Close properties" onClick={close}><X size={18} /></button>
    </div>
    <div className="sheet-body">
      {data.kind !== 'JUNCTION' && <div className="sheet-section" style={{ paddingBottom: 12 }}>
        <StatusPill status={missing.length ? needsStatus(missing.map(item => item.label)) : UI_STATUS.equipment_ready} /></div>}
      {locked && canAct && <section className="sheet-section"><h3>Operate</h3>
        <div className="sheet-actions">{switching ? <>
          <button disabled={busy} onClick={() => operate(node.id, 'open')}>Open</button>
          <button className="primary" disabled={busy} onClick={() => operate(node.id, 'close')}>Close</button>
        </> : trip ? <>
          <button disabled={busy} onClick={() => operate(node.id, 'reset')}>Reset</button>
          <button className="danger" disabled={busy} onClick={() => operate(node.id, 'operate')}>Operate</button>
        </> : <button className="primary" disabled={busy} onClick={() => operate(node.id, 'press')}>
          {data.push_mode === 'latching_status' ? 'Toggle' : 'Press'}</button>}</div>
        {operation && <p className="small" role="status" style={{ margin: 0 }}>{operation}</p>}</section>}
      <section className="sheet-section"><BasicSettings data={data} locked={locked} update={update} /></section>
      {showState && <section className="sheet-section"><h3>{info.state} comes from</h3>
        <StateMappingEditor key={`${node.id}:state:${data.push_mode}`} data={data} ieds={project.ieds}
          disabled={locked} update={update} /></section>}
      {showControl && <section className="sheet-section"><h3>Command goes to</h3>
        <ControlMapping key={`${node.id}:control:${data.push_mode}`} data={data} ieds={project.ieds}
          disabled={locked} update={update} /></section>}
      {data.kind === 'RELAY' && <section className="sheet-section"><h3>Signals on this relay</h3>
        <RelayStatusEditor key={node.id} nodeId={node.id} data={data} project={project} locked={locked}
          values={values} update={update} perform={perform} /></section>}
      {source && <section className="sheet-section"><h3>How the value changes</h3>
        <SourceEditor point={source} project={project} values={values} perform={perform} /></section>}
      <section className="sheet-section"><AdvancedSettings data={data} locked={locked} update={update} applyColors={applyColors} />
        {!locked && <div className="row">
          {!['JUNCTION', 'RELAY'].includes(data.kind) && <button className="ghost" onClick={() =>
            update({ ...data, rotation: (data.rotation + 90) % 360 })}><RotateCw size={16} />Rotate</button>}
          <button className="ghost" onClick={duplicate}><Copy size={16} />Duplicate</button>
          {linkedIed && showFlow && <button className="ghost" onClick={() => showFlow(linkedIed)}>
            <Share2 size={16} />Show signal flow</button>}
          <button className="ghost" style={{ color: 'var(--error)' }} onClick={remove}><Trash2 size={16} />Delete</button>
        </div>}</section>
    </div>
  </aside>;
}
