import { useEffect, useState } from 'react';
import type { EquipmentData, Ied, Signal, SignalBinding } from '../../api/types';
import { SignalSearchPicker } from '../../components/SignalSearchPicker';
import '../../components/signal-search-picker.css';
import { api } from '../../api/client';
import { Link2 } from 'lucide-react';
import { shortRef } from '../../equipment/relayDisplay';

const switches = new Set(['CB', 'DS', 'ES']);
const numeric = ['INT8', 'INT16', 'INT32', 'INT64', 'INT128', 'INT8U', 'INT16U',
  'INT24U', 'INT32U', 'FLOAT32', 'FLOAT64'];
type Choice = 'none' | 'one' | 'contacts';

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

function Pick({ binding, iedName, disabled, clear }: {
  binding: SignalBinding; iedName?: string; disabled: boolean; clear: () => void;
}) {
  const label = useSignalLabel(binding);
  const short = shortRef(binding.ref, iedName);
  const described = !!label && label !== binding.ref;
  return <div className="signal-pick"><span className="signal-pick-mark" aria-hidden="true"><Link2 size={14} /></span>
    <span className="signal-search-text" title={binding.ref}>
      <strong>{described ? label : short}</strong>
      <small>{iedName || 'Unknown IED'}{described ? ` · ${short}` : ''}</small></span>
    {!disabled && <button className="ghost" onClick={clear}>Change</button>}
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
  const [changing, setChanging] = useState<'one' | 'open' | 'closed' | null>(null);
  useEffect(() => {
    if (data.state_mapping) setChoice(
      data.state_mapping.mode === 'contacts_to_dbpos' ? 'contacts' : 'one');
    if (data.state_mapping?.mode === 'contacts_to_dbpos') {
      setOpened(data.state_mapping.open); setClosed(data.state_mapping.closed);
    }
  }, [data.state_mapping]);
  const selectChoice = (next: Choice) => {
    setChoice(next); setOpened(undefined); setClosed(undefined); setChanging(null);
    update({ ...data, state_mapping: null });
  };
  const saveContact = (side: 'open' | 'closed', binding: SignalBinding) => {
    const nextOpen = side === 'open' ? binding : opened;
    const nextClosed = side === 'closed' ? binding : closed;
    setOpened(nextOpen); setClosed(nextClosed); setChanging(null);
    if (nextOpen && nextClosed) update({ ...data, state_mapping: {
      mode: 'contacts_to_dbpos', open: nextOpen, closed: nextClosed,
    } });
  };
  const one = data.state_mapping?.mode === 'direct' || data.state_mapping?.mode === 'single_to_dbpos'
    ? data.state_mapping.source : undefined;
  const isSwitch = switches.has(data.kind);
  const allowed = data.kind === 'TRIP_RELAY' || data.kind === 'PUSH_BUTTON' ? ['BOOLEAN'] :
    data.kind === 'SOURCE' ? numeric : isSwitch ? ['Dbpos', 'BOOLEAN'] : undefined;
  const saveOne = (binding: SignalBinding, signal: Signal) => {
    setChanging(null);
    update({ ...data, state_mapping: signal.kind === 'BOOLEAN' && isSwitch
      ? { mode: 'single_to_dbpos', source: binding } : { mode: 'direct', source: binding } });
  };
  const sourceFc = data.kind === 'SOURCE' ? 'MX' : undefined;
  const iedName = (binding?: SignalBinding) => ieds.find(ied => ied.id === binding?.ied_id)?.name;
  return <div className="stack">
    <label>Connection<select disabled={disabled} value={choice}
      onChange={event => selectChoice(event.target.value as Choice)}>
      <option value="none">Not connected</option>
      <option value="one">{isSwitch ? 'One position point' : 'One IED point'}</option>
      {isSwitch && <option value="contacts">Two points: open and closed contacts</option>}
    </select></label>
    {choice === 'one' && (one && !changing ? <Pick binding={one} iedName={iedName(one)} disabled={disabled} clear={() => setChanging('one')} /> :
      <SignalSearchPicker ieds={ieds} allowedKinds={allowed} allowedFc={sourceFc}
        writableOnly disabled={disabled} choose={saveOne} />)}
    {isSwitch && choice === 'contacts' && <>
      <h4 className="mapping-subhead">Open contact</h4>
      {opened && changing !== 'open' ? <Pick binding={opened} iedName={iedName(opened)} disabled={disabled} clear={() => setChanging('open')} /> :
        <SignalSearchPicker ieds={ieds} allowedKinds={['BOOLEAN']} writableOnly
          disabled={disabled} choose={binding => saveContact('open', binding)} />}
      <h4 className="mapping-subhead">Closed contact</h4>
      {closed && changing !== 'closed' ? <Pick binding={closed} iedName={iedName(closed)} disabled={disabled} clear={() => setChanging('closed')} /> :
        <SignalSearchPicker ieds={ieds} allowedKinds={['BOOLEAN']} writableOnly
          disabled={disabled} choose={binding => saveContact('closed', binding)} />}
    </>}
  </div>;
}
