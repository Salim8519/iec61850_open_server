import { useEffect, useState } from 'react';
import { Link2 } from 'lucide-react';
import '../../components/signal-search-picker.css';
import { shortRef } from '../../equipment/relayDisplay';
import type { EquipmentData, Ied } from '../../api/types';

export function ControlMapping({ data, ieds, disabled, update }: {
  data: EquipmentData; ieds: Ied[]; disabled: boolean; update: (data: EquipmentData) => void;
}) {
  const expected = ['TRIP_RELAY', 'PUSH_BUTTON'].includes(data.kind) ? 'SPC' : 'DPC';
  const onlyIed = ieds.length === 1 ? ieds[0].id : '';
  const [iedId, setIed] = useState(data.control_binding?.ied_id || onlyIed);
  useEffect(() => setIed(data.control_binding?.ied_id || onlyIed), [data.control_binding?.ied_id, onlyIed]);
  const ied = ieds.find(item => item.id === iedId);
  const controls = ied?.controls.filter(control => control.cdc === expected) || [];
  const selected = controls.find(control => control.ref === data.control_binding?.ref);
  const compatibility = data.control_binding?.compatibility || 'standard';
  const noun = expected === 'DPC' ? 'breaker/switch control' : 'command';
  return <div className="stack">
    <label>IED<select disabled={disabled} value={iedId} onChange={event => {
      setIed(event.target.value); update({ ...data, control_binding: null });
    }}><option value="">Choose an IED</option>{ieds.map(item => {
      const count = item.controls.filter(control => control.cdc === expected).length;
      return <option key={item.id} value={item.id} disabled={!count}>
        {item.name} — {count ? `${count} ${noun}${count === 1 ? '' : 's'}` : `no ${noun}s`}</option>;
    })}</select></label>
    {ied && <label>Control object<select disabled={disabled}
      value={data.control_binding?.ied_id === iedId ? data.control_binding.ref : ''}
      onChange={event => {
        const control = controls.find(item => item.ref === event.target.value);
        const autoState = control && !data.state_mapping && data.kind !== 'PUSH_BUTTON';
        update({ ...data, control_binding: control ? { ied_id: iedId, ref: control.ref, compatibility: 'standard' } : null,
          ...(autoState ? { state_mapping: { mode: 'direct' as const,
            source: { ied_id: iedId, ref: control.status, report_ref: null } } } : {}) });
      }}>
      <option value="">Not mapped</option>{controls.map(control =>
        <option key={control.ref} value={control.ref}>{shortRef(control.ref, ied.name)}</option>)}</select></label>}
    {selected?.model === 4 && expected === 'DPC' && data.control_binding && <>
      <label>External client behavior<select disabled={disabled} value={compatibility}
        onChange={event => update({ ...data, control_binding: {
          ...data.control_binding!, compatibility: event.target.value as 'standard' | 'concen_inverted_sbow',
        } })}>
        <option value="standard">Standard enhanced SBO</option>
        <option value="concen_inverted_sbow">CONCEN inverted SBO compatibility</option>
      </select></label>
      {compatibility === 'concen_inverted_sbow' && <p role="status" className="small import-unavailable">
        Non-standard simulation mode: CONCEN Open/Close is inverted and its successful SBOw moves the switch immediately.
      </p>}
    </>}
    {selected && <div className="signal-pick"><span className="signal-pick-mark" aria-hidden="true"><Link2 size={14} /></span>
      <span className="signal-search-text" title={selected.ref}>
      <strong>{shortRef(selected.ref, ied?.name)}</strong>
      <small>{selected.cdc} ready · model {selected.model}{selected.blocked ? ' · blocked' : ''}</small></span>
      {selected.blocked && <span className="badge warning">Blocked</span>}</div>}
    {ied && !controls.length && <p role="status" className="small muted">No compatible {expected} controls in this IED.</p>}
  </div>;
}
