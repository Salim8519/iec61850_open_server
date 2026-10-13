import { Box, ChevronRight, Plus } from 'lucide-react';
import { equipment } from '../../equipment/registry';
import { EquipmentSymbol } from '../../equipment/symbols/EquipmentSymbol';
import type { EquipmentKind, Ied } from '../../api/types';

export function Palette({ locked, ieds, add, openIeds }: {
  locked: boolean; ieds: Ied[]; add: (kind: EquipmentKind) => void; openIeds: () => void;
}) {
  return <aside className="palette">
    <div className="section-label">Equipment library</div>
    <p className="muted small" style={{ marginTop: 8 }}>Drag onto the canvas or click to add.</p>
    <div className="palette-items">
      {equipment.map(item => <button className="palette-item" key={item.kind} draggable={!locked}
        disabled={locked} onClick={() => add(item.kind)}
        onDragStart={event => { event.dataTransfer.setData('application/studio-equipment', item.kind); event.dataTransfer.effectAllowed = 'move'; }}>
        <span className="palette-symbol" data-kind={item.kind}><EquipmentSymbol kind={item.kind} size={30} /></span>
        <span><strong>{item.label}</strong><small>{item.subtitle}</small></span>
      </button>)}
    </div>
    <hr />
    <div className="row" style={{ justifyContent: 'space-between' }}>
      <span className="section-label">Project IEDs</span>
      <button className="icon ghost" aria-label="Manage IEDs" onClick={openIeds}><Plus size={14} /></button>
    </div>
    {ieds.length === 0 ? <p className="muted small">Import your CID or add a template to connect equipment.</p> :
      ieds.map(ied => <button className="ied-list-item" key={ied.id} onClick={openIeds}>
        <Box size={15} /><span>{ied.name}<small>{ied.ip}</small></span><ChevronRight size={12} />
      </button>)}
  </aside>;
}
