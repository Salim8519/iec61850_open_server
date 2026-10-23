import { useState } from 'react';
import { Send } from 'lucide-react';
import type { GooseLeaf } from '../../../api/gooseMessages';
import { leafName, valueWords } from '../../../equipment/gooseLinkChoices';

/** One value of a GOOSE message that can be changed from the IED component's sheet (ADR-0112):
 * On/Off, Open/Closed (a position) or a number box. The current value is shown in words; a button
 * is disabled with the reason when nothing can be sent. Go writes the value; the IED sends GOOSE. */
export function GooseValueSend({ leaf, named, disabled, why, send }: {
  leaf: GooseLeaf; named: boolean; disabled: boolean; why: string; send: (value: boolean | number, words: string) => void;
}) {
  const [text, setText] = useState('');
  const [problem, setProblem] = useState('');
  const name = leafName(leaf.ref);
  const now = valueWords(leaf);
  const title = disabled ? why : '';
  const sendNumber = () => {
    const value = Number(text.trim());
    if (!text.trim() || !Number.isFinite(value)) { setProblem('Type a number.'); return; }
    setProblem(''); send(value, `${name} ${text.trim()}`);
  };
  return <div className="goose-send">
    {/* The member line above already names a one-value member: only its value is said here. */}
    <span className="goose-send-name" title={leaf.description || leaf.ref}>{named ? name : ''}
      {now && <small>{named ? ' · now ' : 'Now '}{now}</small>}</span>
    <span className="goose-send-actions" role="group" aria-label={`Send ${name}`}>
      {leaf.value_type === 'onoff' && <>
        <button disabled={disabled} title={title} onClick={() => send(true, `${name} On`)}>On</button>
        <button disabled={disabled} title={title} onClick={() => send(false, `${name} Off`)}>Off</button></>}
      {leaf.value_type === 'position' && <>
        <button disabled={disabled} title={title} onClick={() => send(1, `${name} Open`)}>Open</button>
        <button disabled={disabled} title={title} onClick={() => send(2, `${name} Closed`)}>Closed</button></>}
      {leaf.value_type === 'number' && <>
        <input value={text} inputMode="decimal" disabled={disabled} aria-label={`New value of ${name}`} placeholder="Number"
          onChange={event => setText(event.target.value)} onKeyDown={event => { if (event.key === 'Enter') sendNumber(); }} />
        <button disabled={disabled} title={title} onClick={sendNumber}><Send size={14} />Send</button></>}
    </span>
    {problem && <p className="relay-error" role="alert">{problem}</p>}
  </div>;
}
