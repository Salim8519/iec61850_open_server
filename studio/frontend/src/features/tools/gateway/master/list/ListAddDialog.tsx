import { X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { MasterProtocol } from '../../../../../api/master';
import type { ListPoint, MasterType } from '../../../../../api/masterList';
import { PROTOCOL_NAMES } from '../../../../../copy/gateway';
import { ADD_LIST_COPY } from '../../../../../copy/masterList';
import { SegmentedControl } from '../../../../../ui/SegmentedControl';
import { ListOnePoint } from './ListOnePoint';
import { ListPaste } from './ListPaste';
import '../../../../ieds/import-selection.css';
import '../../add/add-points.css';

export type ListSource = 'one' | 'range' | 'paste';

/** "Add points" of the SCADA master (ADR-0110), built from the gateway's dialog pieces (ADR-0105):
 *  one point, a range, or rows pasted from Excel / the Excel template / a CSV file. Every way ends in
 *  "Add N points", which hands the engine-checked points to the list (saved at once, with Undo).
 *  There is no IED anywhere: the master only knows addresses and types. Esc or × closes without adding. */
export function ListAddDialog({ protocol, types, start = 'one', add, close }: {
  protocol: MasterProtocol; types: MasterType[]; start?: ListSource; add: (points: ListPoint[]) => void; close: () => void;
}) {
  const [source, setSource] = useState<ListSource>(start);
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => { dialog.current?.showModal(); }, []);
  return <dialog ref={dialog} className="import-dialog add-points" aria-labelledby="master-add-title"
    onCancel={event => { event.preventDefault(); close(); }}>
    <div className="row add-points-head"><h3 id="master-add-title">{ADD_LIST_COPY.title}</h3>
      <span className="small muted">{PROTOCOL_NAMES[protocol]}</span><span className="gateway-spacer" />
      <button type="button" className="ghost icon-button" onClick={close} aria-label={ADD_LIST_COPY.close} title={ADD_LIST_COPY.close}>
        <X size={18} /></button></div>
    <SegmentedControl label={ADD_LIST_COPY.title} value={source} onChange={setSource}
      options={(Object.keys(ADD_LIST_COPY.sources) as ListSource[]).map(key => ({ value: key, label: ADD_LIST_COPY.sources[key],
        title: ADD_LIST_COPY.sourceHints[key] }))} />
    <p className="small muted add-points-hint">{ADD_LIST_COPY.sourceHints[source]}</p>
    <div className="add-points-body">
      {source !== 'paste' && <ListOnePoint key={source} protocol={protocol} types={types} range={source === 'range'} add={add} />}
      {source === 'paste' && <ListPaste protocol={protocol} types={types} add={add} />}
    </div>
  </dialog>;
}
