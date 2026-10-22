import { X } from 'lucide-react';
import { useState } from 'react';
import type { GatewayPoint, PointKind } from '../../../api/gateway';
import type { Ied } from '../../../api/types';
import { SignalSearchPicker } from '../../../components/SignalSearchPicker';
import { CHANGE_COPY } from '../../../copy/gateway';
import { NUMERIC_KINDS } from '../../../equipment/measurementDisplay';

const WHOLE = NUMERIC_KINDS.filter(kind => kind.startsWith('INT'));
/** IEC 61850 value kinds each gateway type may carry (the backend's AllowedKinds, checked again when the list is saved). */
const FITS: Record<PointKind, string[]> = { single: ['BOOLEAN'], double: ['Dbpos'], float: [...NUMERIC_KINDS, 'Enum'],
  integer: [...WHOLE, 'Enum'], command_single: [], command_double: [], step_position: WHOLE, command_step: [],
  command_setpoint: [] };

/** A row under a point that moves it to another IED or path: the signal search for status and
 *  measured points, the controllable objects of an IED for commands. The change is saved with the list.
 *  `taken` ("iedId:ref") are signals other points of this type already serve; they cannot be chosen twice. */
export function SignalChange({ row, ieds, taken, columns, choose, close }: {
  row: GatewayPoint; ieds: Ied[]; taken: Set<string>; columns: number;
  choose: (ied: Ied, ref: string) => void; close: () => void;
}) {
  const [iedId, setIedId] = useState(row.ied);
  const ied = ieds.find(item => item.id === iedId);
  const pick = (id: string, ref: string) => { const found = ieds.find(item => item.id === id); if (found) choose(found, ref); };
  return <tr className="signal-change-row"><td colSpan={columns}><div className="signal-change">
    <div className="row"><strong>{CHANGE_COPY.title(row.address)}</strong>
      <span className="small muted">{CHANGE_COPY.hint}</span><span className="gateway-spacer" />
      <button type="button" className="ghost icon-button" aria-label={CHANGE_COPY.cancel} title={CHANGE_COPY.cancel}
        onClick={close}><X size={15} /></button></div>
    {!ieds.length ? <p className="small muted">{CHANGE_COPY.noIeds}</p>
      : row.kind.startsWith('command')
        ? <div className="row signal-change-command">
          <label>IED<select value={iedId} onChange={event => setIedId(event.target.value)}>
            <option value="">{CHANGE_COPY.chooseIed}</option>
            {ieds.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          {ied && <label>{CHANGE_COPY.control}<select value={ied.id === row.ied ? row.ref : ''}
            onChange={event => event.target.value && pick(ied.id, event.target.value)}>
            <option value="">{ied.controls.length ? CHANGE_COPY.chooseControl : CHANGE_COPY.noControls}</option>
            {ied.controls.map(control => { const used = taken.has(`${ied.id}:${control.ref}`);
              return <option key={control.ref} value={control.ref} disabled={used}>
                {control.ref} · {control.cdc}{used ? ` · ${CHANGE_COPY.used}` : ''}</option>; })}
          </select></label>}</div>
        : <SignalSearchPicker ieds={ieds} allowedKinds={FITS[row.kind]} chooseLabel="Use this" memoryKey="gateway-change" exclude={taken}
          chosen={{ ied_id: row.ied, report_ref: null, ref: row.ref }} choose={binding => pick(binding.ied_id, binding.ref)} />}
  </div></td></tr>;
}
