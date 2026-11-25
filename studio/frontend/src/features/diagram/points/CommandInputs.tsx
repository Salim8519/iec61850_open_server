import { useState } from 'react';
import { Send } from 'lucide-react';
import type { PanelPoint } from '../../../api/types';
import type { CommandControl } from '../../../equipment/pointRequests';
import { enumChoices, parseCommandNumber, rangeHint } from '../../../equipment/typedCommand';
import { SBO_COPY } from '../../../copy/selectOperate';

/** The value inputs of a typed command (ADR-0108), shared by the point panel and Devices (ADR-0149):
 *  a number box (INC, APC) and a list of labels (ENC). `point` names the command (and the unit of a set point). */
type Inputs = { point: Pick<PanelPoint, 'label' | 'unit'>; control?: CommandControl };

const LIST_PLACEHOLDER = 'Choose a value';

/** A typed number (INC whole numbers, APC with the row's unit): Enter or the button picks it. */
export function NumberBox({ point, control, disabled, title, verb, pick, problem }: Inputs & { problem: (text: string) => void;
  disabled: boolean; title: string; verb: string; pick: (value: number) => void;
}) {
  const [text, setText] = useState('');
  const hint = rangeHint(control);
  const apply = () => {
    const result = parseCommandNumber(control, text);
    if (result.value === undefined) { problem(result.error || 'Type a number.'); return; }
    problem(''); pick(result.value);
  };
  const bounds = control as { min?: number | null; max?: number | null; step?: number | null } | undefined;
  return <span className="pp-value-box" title={disabled ? title : hint || SBO_COPY.optionsHint}>
    <input value={text} type="number" inputMode="decimal" disabled={disabled} min={bounds?.min ?? undefined}
      max={bounds?.max ?? undefined} step={bounds?.step ?? 'any'} placeholder={hint.replace(/^from /, '').replace(/,? ?steps of .*$/, '')}
      aria-label={`Value to send: ${point.label}${hint ? ` (${hint})` : ''}`}
      onChange={event => setText(event.target.value)}
      // preventDefault: the same Enter must not also press the question's focused Cancel (change 0176).
      onKeyDown={event => { if (event.key === 'Enter') { event.preventDefault(); apply(); } }} />
    {control?.cdc === 'APC' && point.unit && <span className="pp-unit">{point.unit}</span>}
    <button type="button" className="wire" disabled={disabled} onClick={apply}><Send size={13} aria-hidden="true" />{verb}</button>
  </span>;
}

/** An ENC command: one of the model's labels, then the button. Nothing is picked at first: a command
 *  row has no live value, so the first label ("off") must not look like the present one (change 0176). */
export function ListBox({ point, control, disabled, title, verb, pick }: Inputs & {
  disabled: boolean; title: string; verb: string; pick: (value: string) => void;
}) {
  const labels = enumChoices(control);
  const [chosen, setChosen] = useState('');
  if (!labels.length) return <span className="small muted">This IED's model gives no values for this list.</span>;
  return <span className="pp-value-box" title={title}>
    <select value={chosen} disabled={disabled} aria-label={`Value to send: ${point.label}`}
      onChange={event => setChosen(event.target.value)}>
      <option value="" disabled>{LIST_PLACEHOLDER}</option>
      {labels.map(label => <option key={label} value={label}>{label}</option>)}
    </select>
    <button type="button" className="wire" disabled={disabled || !chosen} onClick={() => pick(chosen)}><Send size={13} aria-hidden="true" />{verb}</button>
  </span>;
}

