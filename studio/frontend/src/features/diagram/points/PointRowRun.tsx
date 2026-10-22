import { useEffect, useState } from 'react';
import { Check } from 'lucide-react';
import type { PanelPoint } from '../../../api/types';
import { displayToRaw, positionWord, rawToInput } from '../../../equipment/pointDisplay';
import { DOUBLE_POSITIONS } from '../../../equipment/pointPanel';
import type { CommandChoice, CommandControl, CommandValue } from '../../../equipment/pointRequests';
import { POINT_COPY } from '../../../copy/pointPanel';
import { PointCommand } from './PointCommand';

type Props = {
  nodeId: string; point: PanelPoint; control?: CommandControl; raw: unknown; disabled: boolean; why: string;
  set: (value: boolean | number, done: string) => void; ask: (choice: CommandChoice) => void;
  send: (point: PanelPoint, value: CommandValue, done: string) => Promise<void>;
  problem: (text: string) => void;
};

/** A button that says what it does; the current value of the point is marked with a tick and the word,
 *  not with colour alone. */
function Choice({ label, current, disabled, title, onClick }: {
  label: string; current?: boolean; disabled: boolean; title: string; onClick: () => void;
}) {
  return <button type="button" className={`pp-choice${current ? ' current' : ''}`} aria-pressed={current} disabled={disabled}
    title={title} onClick={onClick}>{current && <Check size={13} aria-hidden="true" />}{label}</button>;
}

/** A value point: type the value as shown (with its unit beside the box), Enter sets it. */
function ValueBox({ point, raw, disabled, title, set, problem }: Pick<Props, 'point' | 'raw' | 'set' | 'problem'> & {
  disabled: boolean; title: string;
}) {
  const [text, setText] = useState('');
  const [typing, setTyping] = useState(false);
  useEffect(() => { if (!typing) setText(rawToInput(point, raw)); }, [point, raw, typing]);
  const apply = () => {
    const result = displayToRaw(point, text);
    if (result.value === undefined) { problem(result.error || 'Type a number.'); return; }
    problem(''); setTyping(false); set(result.value, `set to ${text.trim()}${point.unit ? ` ${point.unit}` : ''}`);
  };
  return <span className="pp-value-box" title={disabled ? title : POINT_COPY.valueHint}>
    <input value={text} inputMode="decimal" disabled={disabled} aria-label={`${POINT_COPY.valueInput}: ${point.label}`}
      onFocus={() => setTyping(true)} onBlur={() => setTyping(false)} onChange={event => setText(event.target.value)}
      onKeyDown={event => { if (event.key === 'Enter') apply(); if (event.key === 'Escape') { setTyping(false); (event.target as HTMLInputElement).blur(); } }} />
    {point.unit && <span className="pp-unit">{point.unit}</span>}
    <button type="button" disabled={disabled} onClick={apply} title={POINT_COPY.valueHint}>{POINT_COPY.valueApply}</button>
  </span>;
}

/** What can be done to one point while the IEDs run. Everything is disabled with a reason when it cannot. */
export function PointRowRun({ nodeId, point, control, raw, disabled, why, set, ask, send, problem }: Props) {
  const title = disabled ? why : '';
  if (point.type === 'status') {
    return <span className="pp-actions" role="group" aria-label={`Set ${point.label}`}>
      <Choice label="ON" current={raw === true} disabled={disabled} title={title || 'Set this point ON'} onClick={() => set(true, 'set ON')} />
      <Choice label="OFF" current={raw === false} disabled={disabled} title={title || 'Set this point OFF'} onClick={() => set(false, 'set OFF')} />
    </span>;
  }
  if (point.type === 'double') {
    return <span className="pp-actions" role="group" aria-label={`Set ${point.label}`}>
      {DOUBLE_POSITIONS.map(item => <Choice key={item.raw} label={item.word} current={raw === item.raw} disabled={disabled}
        title={title || `Set this point to ${positionWord(item.raw)}`} onClick={() => set(item.raw, `set to ${item.word}`)} />)}
    </span>;
  }
  if (point.type === 'value') return <ValueBox point={point} raw={raw} disabled={disabled} title={title} set={set} problem={problem} />;
  return <PointCommand nodeId={nodeId} point={point} control={control} disabled={disabled} why={why} ask={ask} send={send}
    problem={problem} />;
}
