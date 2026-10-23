import { useEffect, useState } from 'react';
import type { EquipmentData, Ied, Signal, SignalBinding, StateMapping } from '../../api/types';
import { SignalSearchPicker } from '../../components/SignalSearchPicker';
import '../../components/signal-search-picker.css';
import { api } from '../../api/client';
import { Link2 } from 'lucide-react';
import { shortRef } from '../../equipment/relayDisplay';

const switches = new Set(['CB', 'DS', 'ES']);
const numeric = ['INT8', 'INT16', 'INT32', 'INT64', 'INT128', 'INT8U', 'INT16U',
  'INT24U', 'INT32U', 'FLOAT32', 'FLOAT64'];
type Choice = 'none' | 'one' | 'contacts';
type Slot = 'one' | 'open' | 'closed';

function useSignalLabel(binding?: SignalBinding): string {
  const [label, setLabel] = useState(binding?.ref || '');
  useEffect(() => {
    if (!binding) { setLabel(''); return; }
    let cancelled = false;
    api<{ items: Signal[] }>(`/ieds/${binding.ied_id}/signals?ref=${encodeURIComponent(binding.ref)}`)
      .then(result => { if (!cancelled) setLabel(result.items[0]?.description || binding.ref); });
    return () => { cancelled = true; };
  }, [binding?.ied_id, binding?.ref]);
  return label;
}

/** The chosen point. "Change" opens the picker where the user left it; "Done" closes it. */
function Pick({ binding, iedName, disabled, open, toggle, undo }: {
  binding: SignalBinding; iedName?: string; disabled: boolean; open: boolean;
  toggle: () => void; undo?: () => void;
}) {
  const label = useSignalLabel(binding);
  const short = shortRef(binding.ref, iedName);
  const described = !!label && label !== binding.ref;
  return <div className="signal-pick"><span className="signal-pick-mark" aria-hidden="true"><Link2 size={14} /></span>
    <span className="signal-search-text" title={binding.ref}>
      <strong>{described ? label : short}</strong>
      <small>{iedName || 'Unknown IED'}{described ? ` · ${short}` : ''}</small></span>
    {!disabled && <span className="signal-pick-actions">
      {undo && <button className="ghost" onClick={undo}>Undo</button>}
      <button className="ghost" onClick={toggle}>{open ? 'Done' : 'Change'}</button></span>}
  </div>;
}

export function StateMappingEditor({ data, ieds, disabled, update }: {
  data: EquipmentData; ieds: Ied[]; disabled: boolean; update: (data: EquipmentData) => void;
}) {
  const [choice, setChoice] = useState<Choice>(
    !data.state_mapping ? 'none' : data.state_mapping.mode === 'contacts_to_dbpos' ? 'contacts' : 'one');
  const contacts = data.state_mapping?.mode === 'contacts_to_dbpos' ? data.state_mapping : null;
  const [opened, setOpened] = useState<SignalBinding | undefined>(contacts?.open);
  const [closed, setClosed] = useState<SignalBinding | undefined>(contacts?.closed);
  // The picker that stays open after a choice, so a wrong point is replaced with one click.
  const [changing, setChanging] = useState<Slot | null>(null);
  // The connection before the latest choice, for one-click Undo while the picker is open.
  const [previous, setPrevious] = useState<{ value: StateMapping | null } | null>(null);
  useEffect(() => {
    if (data.state_mapping) setChoice(
      data.state_mapping.mode === 'contacts_to_dbpos' ? 'contacts' : 'one');
    if (data.state_mapping?.mode === 'contacts_to_dbpos') {
      setOpened(data.state_mapping.open); setClosed(data.state_mapping.closed);
    }
  }, [data.state_mapping]);
  // Switching between one point and two contacts keeps the saved connection until the new one
  // is complete; only "Not connected" removes it.
  const selectChoice = (next: Choice) => {
    setChoice(next); setChanging(null); setPrevious(null);
    if (next === 'none') { setOpened(undefined); setClosed(undefined); update({ ...data, state_mapping: null }); }
  };
  const remember = () => { if (!previous) setPrevious({ value: data.state_mapping ?? null }); };
  const saveContact = (side: 'open' | 'closed', binding: SignalBinding) => {
    const nextOpen = side === 'open' ? binding : opened;
    const nextClosed = side === 'closed' ? binding : closed;
    setOpened(nextOpen); setClosed(nextClosed);
    if (nextOpen && nextClosed) {
      remember();
      update({ ...data, state_mapping: { mode: 'contacts_to_dbpos', open: nextOpen, closed: nextClosed } });
    }
  };
  const one = data.state_mapping?.mode === 'direct' || data.state_mapping?.mode === 'single_to_dbpos'
    ? data.state_mapping.source : undefined;
  const isSwitch = switches.has(data.kind);
  const allowed = data.kind === 'TRIP_RELAY' || data.kind === 'PUSH_BUTTON' ? ['BOOLEAN'] :
    data.kind === 'SOURCE' ? numeric : isSwitch ? ['Dbpos', 'BOOLEAN'] : undefined;
  const saveOne = (binding: SignalBinding, signal: Signal) => {
    setChanging('one');
    remember();
    update({ ...data, state_mapping: signal.kind === 'BOOLEAN' && isSwitch
      ? { mode: 'single_to_dbpos', source: binding } : { mode: 'direct', source: binding } });
  };
  const undo = previous ? () => {
    const value = previous.value;
    setPrevious(null); setChanging(null);
    if (value?.mode === 'contacts_to_dbpos') { setOpened(value.open); setClosed(value.closed); }
    update({ ...data, state_mapping: value });
  } : undefined;
  const toggle = (slot: Slot) => () => {
    if (changing === slot) { setChanging(null); setPrevious(null); } else setChanging(slot);
  };
  const sourceFc = data.kind === 'SOURCE' ? 'MX' : undefined;
  const iedName = (binding?: SignalBinding) => ieds.find(ied => ied.id === binding?.ied_id)?.name;
  const keeping = !!data.state_mapping && !(choice === 'contacts' ? contacts : one);
  // The note that says why a type is chosen names the slot (ADR-0113).
  const slotName = (key: Slot) => key === 'open' ? 'Open contact' : key === 'closed' ? 'Closed contact' :
    isSwitch ? 'Position' : data.kind === 'SOURCE' ? 'Value' : 'Point';
  const slot = (key: Slot, binding: SignalBinding | undefined, kinds: string[] | undefined,
    choose: (binding: SignalBinding, signal: Signal) => void) => <>
    {binding && <Pick binding={binding} iedName={iedName(binding)} disabled={disabled}
      open={changing === key} toggle={toggle(key)} undo={changing === key ? undo : undefined} />}
    {(!binding || changing === key) && <SignalSearchPicker ieds={ieds} allowedKinds={kinds}
      allowedFc={key === 'one' ? sourceFc : undefined} writableOnly disabled={disabled}
      memoryKey={`sld-state-${key}`} chosen={binding} near={data.label} slot={slotName(key)} choose={choose} />}
  </>;
  return <div className="stack">
    <label>Connection<select disabled={disabled} value={choice}
      onChange={event => selectChoice(event.target.value as Choice)}>
      <option value="none">Not connected</option>
      <option value="one">{isSwitch ? 'One position point' : 'One IED point'}</option>
      {isSwitch && <option value="contacts">Two points: open and closed contacts</option>}
    </select></label>
    {keeping && <p className="small muted" style={{ margin: 0 }}>
      The current connection stays until you choose the new point{choice === 'contacts' ? 's' : ''}.</p>}
    {choice === 'one' && slot('one', one, allowed, saveOne)}
    {isSwitch && choice === 'contacts' && <>
      <h4 className="mapping-subhead">Open contact</h4>
      {slot('open', opened, ['BOOLEAN'], binding => { setChanging('open'); saveContact('open', binding); })}
      <h4 className="mapping-subhead">Closed contact</h4>
      {slot('closed', closed, ['BOOLEAN'], binding => { setChanging('closed'); saveContact('closed', binding); })}
    </>}
  </div>;
}
