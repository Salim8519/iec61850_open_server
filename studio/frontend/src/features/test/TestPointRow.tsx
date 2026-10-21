import { ArrowDown, ArrowUp, Trash2 } from 'lucide-react';
import type { Signal, TestControl } from '../../api/types';
import { shortRef } from '../../equipment/relayDisplay';
import type { Kind } from '../../points/pointKind';
import { useSignalValue } from '../../state/useLive';
import { Disclosure } from '../../ui/Disclosure';
import { EditableText } from '../../ui/EditableText';
import { kindRow } from '../../ui/KindBadge';
import { TypeCell } from '../../components/TypeCell';
import { ValueControl } from './ValueControl';
import './test-points.css';

/** A saved test point plus what the engine adds: its signal and whether a generator drives it. */
export type Item = TestControl & { signal: Signal; automatic: boolean; overridden: boolean };
type Perform = (task: () => Promise<unknown>) => Promise<void>;

/** One test point's control; only this cell re-renders when the point's live value changes. */
function PointControl({ item, running, perform }: { item: Item; running: boolean; perform: Perform }) {
  const live = useSignalValue(`${item.ied_id}:${item.ref}`);
  return <ValueControl point={item} signal={item.signal} running={running} perform={perform}
    value={live ?? item.signal.value} segmented generator={item.automatic} driven={item.automatic && !item.overridden}
    pulse={{ active: item.active_value, durationMs: item.duration_ms }} />;
}

/** One line per test point: type, name (click to rename), where it lives, its value, order and remove. */
export function TestPointRow({ item, entry, kind, iedName, running, perform, edit, move, remove }: {
  item: Item; entry: TestControl; kind: Kind; iedName?: string; running: boolean; perform: Perform;
  edit: (entry: TestControl) => void; move: { up?: () => void; down?: () => void }; remove: () => void;
}) {
  return <tr {...kindRow(kind)}>
    <TypeCell kind={kind} raw={item.signal.kind} fc={item.signal.fc} />
    <td style={{ minWidth: 240 }}>
      <div className="test-point-name">
        <EditableText value={item.label} label="Test point name" disabled={running} onSave={text =>
          text.trim() && edit({ ...entry, label: text.trim().slice(0, 100) })} />
        <span className="small muted" title={item.ref}>{iedName || 'Missing device'} · {shortRef(item.ref, iedName)}</span>
      </div>
      {!running && item.signal.kind === 'BOOLEAN' && <Disclosure title="Pulse settings"><div className="form-grid">
        <label>Pulse value<select value={String(item.active_value)} onChange={event => edit({ ...entry, active_value: event.target.value === 'true' })}>
          <option value="true">True</option><option value="false">False</option></select></label>
        <label>Pulse length (ms)<input type="number" min={100} max={60000} step={100} defaultValue={item.duration_ms}
          onBlur={event => Number(event.target.value) !== item.duration_ms && edit({ ...entry, duration_ms: Number(event.target.value) })} /></label>
      </div></Disclosure>}</td>
    <td><PointControl item={item} running={running} perform={perform} /></td>
    <td><div className="row" style={{ justifyContent: 'flex-end', gap: 2 }}>
      <button className="icon ghost" aria-label={`Move ${item.label} up`} disabled={running || !move.up} onClick={move.up}><ArrowUp size={16} /></button>
      <button className="icon ghost" aria-label={`Move ${item.label} down`} disabled={running || !move.down} onClick={move.down}><ArrowDown size={16} /></button>
      <button className="icon ghost" aria-label={`Remove ${item.label}`} disabled={running} onClick={remove}><Trash2 size={16} /></button></div></td>
  </tr>;
}
