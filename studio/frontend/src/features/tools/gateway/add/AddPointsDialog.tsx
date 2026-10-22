import { X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { GatewayPoint, Protocol } from '../../../../api/gateway';
import type { Ied } from '../../../../api/types';
import { PROTOCOL_NAMES } from '../../../../copy/gateway';
import { ADD_COPY } from '../../../../copy/gatewayAdd';
import { SegmentedControl } from '../../../../ui/SegmentedControl';
import { AddOwnPoint } from '../AddOwnPoint';
import { FromIed } from './FromIed';
import { OwnRange } from './OwnRange';
import { PasteRows } from './PasteRows';
import { useTypes } from './useTypes';
import '../../../ieds/import-selection.css';
import './add-points.css';

type Source = 'ied' | 'simulated' | 'paste';
type Simulated = 'one' | 'range';

/** The one "Add points" dialog (ADR-0105). First choice: From an IED, Simulated (made up) or Paste from Excel.
 *  Every way ends in "Add N points", which hands the new points to the list (saved at once, with Undo).
 *  The engine plans and checks; this dialog only shows what it answers. Esc or × closes it without adding. */
export function AddPointsDialog({ protocol, ieds, rows, add, close, fail }: {
  protocol: Protocol; ieds: Ied[]; rows: GatewayPoint[]; add: (points: GatewayPoint[]) => void; close: () => void;
  fail: (message: string) => void;
}) {
  const [source, setSource] = useState<Source>(ieds.length ? 'ied' : 'simulated');
  const [simulated, setSimulated] = useState<Simulated>('one');
  const types = useTypes(protocol, fail);
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => { dialog.current?.showModal(); }, []); // the app's modal dialog, as Devices › Delete
  return <dialog ref={dialog} className="import-dialog add-points" aria-labelledby="add-points-title"
    onCancel={event => { event.preventDefault(); close(); }}>
      <div className="row add-points-head"><h3 id="add-points-title">{ADD_COPY.title}</h3>
        <span className="small muted">{PROTOCOL_NAMES[protocol]}</span><span className="gateway-spacer" />
        <button type="button" className="ghost icon-button" onClick={close} aria-label={ADD_COPY.close} title={ADD_COPY.close}>
          <X size={18} /></button></div>
      <SegmentedControl label={ADD_COPY.title} value={source} onChange={setSource} options={(Object.keys(ADD_COPY.sources) as Source[])
        .map(key => ({ value: key, label: ADD_COPY.sources[key], title: ADD_COPY.sourceHints[key] }))} />
      <p className="small muted add-points-hint">{ADD_COPY.sourceHints[source]}</p>
      <div className="add-points-body">
        {source === 'ied' && <FromIed protocol={protocol} ieds={ieds} rows={rows} types={types} add={add} fail={fail} />}
        {source === 'simulated' && <>
          <SegmentedControl label={ADD_COPY.sources.simulated} value={simulated} onChange={setSimulated}
            options={[{ value: 'one', label: ADD_COPY.one }, { value: 'range', label: ADD_COPY.range, title: ADD_COPY.rangeHint }]} />
          {simulated === 'one'
            ? <AddOwnPoint bare protocol={protocol} rows={rows} fail={fail} close={close} add={point => add([point])} />
            : <OwnRange protocol={protocol} rows={rows} types={types} add={add} fail={fail} />}</>}
        {source === 'paste' && <PasteRows protocol={protocol} rows={rows} add={add} fail={fail} />}
      </div>
  </dialog>;
}
