import { useState, type ReactNode } from 'react';
import { Send } from 'lucide-react';
import { api } from '../../../api/client';
import type { PanelPoint } from '../../../api/types';
import { cancelBody, commandChoices, pointRoute, selectBody, type CommandChoice, type CommandControl, type CommandValue }
  from '../../../equipment/pointRequests';
import { isSbo } from '../../../equipment/selectOperate';
import { commandInput, commandWords, enumChoices, parseCommandNumber, rangeHint } from '../../../equipment/typedCommand';
import { SBO_COPY } from '../../../copy/selectOperate';
import { SboCountdown } from '../sbo/SboCountdown';
import { StepButtons } from '../sbo/StepButtons';
import { useSelectOperate } from '../sbo/useSelectOperate';
import { useSboSteps } from '../sbo/useSboSteps';

type Props = {
  nodeId: string; point: PanelPoint; control?: CommandControl; disabled: boolean; why: string;
  /** A direct command asks one small question first (the row shows it). */
  ask: (choice: CommandChoice) => void;
  send: (point: PanelPoint, value: CommandValue, done: string) => Promise<void>;
  problem: (text: string) => void;
};

const LIST_PLACEHOLDER = 'Choose a value';

/** A typed number (INC whole numbers, APC with the row's unit): Enter or the button picks it. */
function NumberBox({ point, control, disabled, title, verb, pick, problem }: Pick<Props, 'point' | 'control' | 'problem'> & {
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
    <button type="button" disabled={disabled} onClick={apply}><Send size={13} aria-hidden="true" />{verb}</button>
  </span>;
}

/** An ENC command: one of the model's labels, then the button. Nothing is picked at first: a command
 *  row has no live value, so the first label ("off") must not look like the present one (change 0176). */
function ListBox({ point, control, disabled, title, verb, pick }: Pick<Props, 'point' | 'control'> & {
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
    <button type="button" disabled={disabled || !chosen} onClick={() => pick(chosen)}><Send size={13} aria-hidden="true" />{verb}</button>
  </span>;
}

/** What a command row offers while the IEDs run (ADR-0108). A command asks once and sends (an SBO
 *  control is selected and operated in one request, change 0173); with "step by step" on, an SBO
 *  control selects first and shows the countdown with Operate and Cancel below the row. */
export function PointCommand({ nodeId, point, control, disabled, why, ask, send, problem }: Props) {
  const steps = useSboSteps();
  const sbo = isSbo(control?.model) && steps;
  const words = (value: CommandValue) => commandWords(point, control, value);
  const flow = useSelectOperate<CommandValue>({
    select: value => api<{ expires_ms: number }>(pointRoute(nodeId), 'POST', selectBody(point, value)),
    operate: value => send(point, value, words(value)),
    cancel: () => api(pointRoute(nodeId), 'POST', cancelBody(point)),
  });
  const held = flow.state.phase !== 'idle';
  const off = disabled || held;
  const title = disabled ? why : sbo ? SBO_COPY.sboHint : isSbo(control?.model) ? SBO_COPY.sboOneClick : '';
  const verb = sbo ? 'Select' : 'Send';
  const pick = (value: CommandValue) => {
    flow.clear();
    if (sbo) void flow.select(value); else ask({ label: words(value), value });
  };
  const input = commandInput(control);
  let body: ReactNode;
  if (input === 'none') body = <span className="small muted">This IED has no such command any more.</span>;
  else if (input === 'number') body = <NumberBox point={point} control={control} disabled={off} title={title} verb={verb} pick={pick} problem={problem} />;
  else if (input === 'list') body = <ListBox point={point} control={control} disabled={off} title={title} verb={verb} pick={pick} />;
  else if (input === 'steps') body = <StepButtons disabled={off} title={title} label={`${verb} ${point.label}`} press={pick} />;
  else {
    body = <span className="pp-actions" role="group" aria-label={`Send ${point.label}`}>
      {commandChoices(control).map(choice => <button key={choice.label} type="button" className="pp-choice" disabled={off}
        title={title || `${choice.label} to this command`} onClick={() => pick(choice.value)}>
        {sbo ? choice.label.replace(/^Send /, 'Select ') : choice.label}</button>)}
    </span>;
  }
  return <>{body}<SboCountdown flow={flow} what={words} compact /></>;
}
