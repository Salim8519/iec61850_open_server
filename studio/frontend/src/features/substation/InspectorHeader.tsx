import { Check, CircleAlert, Copy, RotateCw, Trash2 } from 'lucide-react';
import type { EquipmentData } from '../../api/types';
import { EquipmentSymbol } from '../../equipment/symbols/EquipmentSymbol';
import { kindInfo, setupItems } from '../../equipment/kindInfo';

type Props = {
  data: EquipmentData; locked: boolean; rotatable: boolean;
  rotate: () => void; duplicate: () => void; remove: () => void;
};

/** Identity, setup checklist and object actions for the selected equipment. */
export function InspectorHeader({ data, locked, rotatable, rotate, duplicate, remove }: Props) {
  const info = kindInfo(data.kind);
  const items = setupItems(data);
  const missing = items.filter(item => !item.done).length;
  return <div className={`inspector-head cat-${info.category}`}>
    <div className="inspector-head-main">
      <span className="inspector-head-symbol"><EquipmentSymbol kind={data.kind} size={34} /></span>
      <div className="inspector-head-title">
        <h2 title={data.label}>{data.label}</h2>
        <span>{info.name} · {info.code}</span>
      </div>
      {!locked && <div className="inspector-head-actions">
        {rotatable && <button className="icon ghost" title="Rotate 90°" aria-label="Rotate 90 degrees"
          onClick={rotate}><RotateCw size={15} /></button>}
        <button className="icon ghost" title="Duplicate" aria-label="Duplicate equipment"
          onClick={duplicate}><Copy size={15} /></button>
        <button className="icon ghost inspector-delete" title="Delete" aria-label="Delete equipment"
          onClick={remove}><Trash2 size={15} /></button>
      </div>}
    </div>
    {items.length > 0 && <div className={`setup ${missing ? 'needed' : 'ready'}`} role="status">
      <strong>{missing ? <><CircleAlert size={14} />Setup needed</> : <><Check size={14} />Ready</>}</strong>
      <ul>{items.map(item => <li key={item.label} className={item.done ? 'done' : 'todo'}>
        {item.done ? <Check size={12} /> : <span className="todo-mark" aria-hidden="true" />}
        {item.label}<span className="sr-only">{item.done ? ' done' : ' missing'}</span></li>)}</ul>
    </div>}
  </div>;
}
