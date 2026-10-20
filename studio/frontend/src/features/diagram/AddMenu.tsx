import { Plus } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { EquipmentKind } from '../../api/types';
import { EquipmentSymbol } from '../../equipment/symbols/EquipmentSymbol';
import { kindInfo } from '../../equipment/kindInfo';
import { BAY_ORDER, BAYS, type BayKind } from '../../equipment/bayTemplates';

const GROUPS: { title: string; kinds: EquipmentKind[] }[] = [
  { title: 'Switching', kinds: ['CB', 'DS', 'ES'] },
  { title: 'Measuring', kinds: ['CT', 'VT', 'SOURCE', 'MEASUREMENT_LIST'] },
  { title: 'Protection & control', kinds: ['RELAY', 'TRIP_RELAY', 'PUSH_BUTTON'] },
  { title: 'Power in and drawing', kinds: ['INFEED', 'JUNCTION'] },
];

/** Big labelled tiles grouped by purpose; click to add or drag onto the canvas. */
export function AddMenu({ disabled, add, addBay, attachHint }: {
  disabled: boolean; add: (kind: EquipmentKind) => void; addBay: (kind: BayKind) => void; attachHint: string;
}) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const close = (event: MouseEvent | KeyboardEvent) => {
      if (event instanceof KeyboardEvent ? event.key === 'Escape' : !root.current?.contains(event.target as Node)) setOpen(false);
    };
    window.addEventListener('mousedown', close); window.addEventListener('keydown', close);
    return () => { window.removeEventListener('mousedown', close); window.removeEventListener('keydown', close); };
  }, [open]);
  return <div ref={root} style={{ position: 'relative' }}>
    <button className="primary" disabled={disabled} aria-expanded={open} onClick={() => setOpen(!open)}><Plus size={17} />Add</button>
    {open && <div className="popover add-menu" style={{ left: 0, top: 'calc(100% + 10px)' }}>
      {GROUPS.map(group => <div key={group.title}><div className="section-label" style={{ padding: '6px 6px 8px' }}>{group.title}</div>
        <div className="add-tiles">{group.kinds.map(kind => <button key={kind} className="add-tile" draggable
          onDragStart={event => { event.dataTransfer.setData('application/studio-equipment', kind); event.dataTransfer.effectAllowed = 'move'; }}
          onClick={() => { add(kind); setOpen(false); }}>
          <EquipmentSymbol kind={kind} size={40} /><span>{kindInfo(kind).name}</span></button>)}</div></div>)}
      <div><div className="section-label" style={{ padding: '6px 6px 8px' }}>Whole bays</div>
        <p className="small muted" style={{ margin: '0 6px 8px' }}>{attachHint}</p>
        <div className="add-tiles">{BAY_ORDER.map(kind => <button key={kind} className="add-tile" draggable title={BAYS[kind].hint}
          onDragStart={event => { event.dataTransfer.setData('application/studio-bay', kind); event.dataTransfer.effectAllowed = 'move'; }}
          onClick={() => { addBay(kind); setOpen(false); }}>
          <EquipmentSymbol kind={kind === 'transformer_bay' ? 'INFEED' : 'CB'} infeed="transformer" size={40} />
          <span>{BAYS[kind].name}</span></button>)}</div></div>
    </div>}
  </div>;
}
