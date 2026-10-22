import { useEffect, useState } from 'react';
import { Link2 } from 'lucide-react';
import type { ControlBinding, Ied, Signal, SignalBinding } from '../../../api/types';
import { SignalSearchPicker } from '../../../components/SignalSearchPicker';
import { shortRef } from '../../../equipment/relayDisplay';

type Choice = { ref: string; cdc: string };

/** One signal of the tap changer: what is chosen, then Change / Remove; Change opens the shared search picker
 *  limited to the kinds and functional constraint that fit. */
export function TapSignalField({ title, hint, ieds, value, kinds, fc, memoryKey, disabled, change }: {
  title: string; hint: string; ieds: Ied[]; value: SignalBinding | null; kinds: string[]; fc: string;
  memoryKey: string; disabled: boolean; change: (binding: SignalBinding | null, signal?: Signal) => void;
}) {
  const [picking, setPicking] = useState(false);
  const ied = value ? ieds.find(item => item.id === value.ied_id) : undefined;
  return <div className="tap-binding">
    <div className="tap-binding-head"><strong>{title}</strong>
      {!disabled && <span className="row">
        <button className="ghost" onClick={() => setPicking(!picking)}>{picking ? 'Done' : value ? 'Change' : 'Choose'}</button>
        {value && <button className="ghost" onClick={() => change(null)}>Remove</button>}</span>}</div>
    {value ? <div className="signal-pick"><span className="signal-pick-mark" aria-hidden="true"><Link2 size={14} /></span>
      <span className="signal-search-text" title={value.ref}><strong>{shortRef(value.ref, ied?.name)}</strong>
        <small>{ied?.name ?? value.ied_id}</small></span></div> : <p className="small muted" style={{ margin: 0 }}>{hint}</p>}
    {picking && !disabled && <SignalSearchPicker ieds={ieds} allowedKinds={kinds} allowedFc={fc} memoryKey={memoryKey}
      chosen={value ?? undefined} choose={(binding, signal) => { change(binding, signal); setPicking(false); }} />}
  </div>;
}

/** One control of the tap changer: IED, then one of the controls of the given classes. */
export function TapControlField({ label, ieds, value, controlsOf, empty, disabled, change }: {
  label: string; ieds: Ied[]; value: ControlBinding | null; controlsOf: (ied: Ied | undefined) => Choice[];
  empty: string; disabled: boolean; change: (binding: ControlBinding | null) => void;
}) {
  const onlyIed = ieds.length === 1 ? ieds[0].id : '';
  const [iedId, setIed] = useState(value?.ied_id || onlyIed);
  useEffect(() => setIed(value?.ied_id || onlyIed), [value?.ied_id, onlyIed]);
  const ied = ieds.find(item => item.id === iedId);
  const controls = controlsOf(ied);
  return <div className="stack">
    <label>{label} · IED<select disabled={disabled} value={iedId} onChange={event => { setIed(event.target.value); change(null); }}>
      <option value="">Choose an IED</option>
      {ieds.map(item => <option key={item.id} value={item.id}>{item.name} — {controlsOf(item).length} fitting</option>)}
    </select></label>
    {ied && <label>{label}<select disabled={disabled} value={value?.ied_id === iedId ? value.ref : ''}
      onChange={event => change(event.target.value ? { ied_id: iedId, ref: event.target.value } : null)}>
      <option value="">Not mapped</option>
      {controls.map(control => <option key={control.ref} value={control.ref}>
        {shortRef(control.ref, ied.name)} · {control.cdc}</option>)}</select></label>}
    {ied && !controls.length && <p role="status" className="small muted" style={{ margin: 0 }}>{empty}</p>}
  </div>;
}

/** A number setting that is saved when it is a valid number and the field is left (or Enter is pressed). */
export function TapNumber({ label, unit, hint, value, whole, disabled, check, change }: {
  label: string; unit?: string; hint?: string; value: number; whole?: boolean; disabled: boolean;
  check?: (next: number) => string | null; change: (next: number) => void;
}) {
  const [text, setText] = useState(String(value));
  useEffect(() => setText(String(value)), [value]);
  const parsed = Number(text);
  const problem = text.trim() === '' || !Number.isFinite(parsed) ? 'Enter a number.' :
    whole && !Number.isInteger(parsed) ? 'Enter a whole number.' : check?.(parsed) ?? null;
  const commit = () => { if (!problem && parsed !== value) change(parsed); };
  return <label>{label}{unit ? ` (${unit})` : ''}
    <input type="number" disabled={disabled} value={text} step={whole ? 1 : 'any'} aria-invalid={!!problem}
      onChange={event => setText(event.target.value)} onBlur={commit}
      onKeyDown={event => { if (event.key === 'Enter') commit(); }} />
    {problem ? <small role="alert" style={{ color: 'var(--error)' }}>{problem}</small> :
      hint ? <small className="muted">{hint}</small> : null}</label>;
}
