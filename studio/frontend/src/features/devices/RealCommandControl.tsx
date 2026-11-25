import { Send, TriangleAlert } from 'lucide-react';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import type { CommandOptions } from '../../api/types';
import { COMMAND_COPY } from '../../copy/commands';
import { GATEWAY_COPY } from '../../copy/gateway';
import { SBO_COPY } from '../../copy/selectOperate';
import { commandChoices, type CommandControl, type CommandValue } from '../../equipment/pointRequests';
import { isSbo } from '../../equipment/selectOperate';
import { commandInput, commandWords } from '../../equipment/typedCommand';
import { ListBox, NumberBox } from '../diagram/points/CommandInputs';
import { SboCountdown } from '../diagram/sbo/SboCountdown';
import { StepButtons } from '../diagram/sbo/StepButtons';
import { useSboSteps } from '../diagram/sbo/useSboSteps';
import { useSelectOperate } from '../diagram/sbo/useSelectOperate';
import { answerText, sendStep, type StepAction } from './realCommandSteps';

type Pending = { action: 'select' | 'operate'; label: string; value: CommandValue };

/** The buttons or value box of one command of a real IED (ADR-0149), used in the Commands pop-up and in
 *  the Data points rows. Every send asks once, in place, so the list never jumps. One click (default)
 *  selects and operates in one go; with "step by step" on, an SBO command selects first and shows the
 *  countdown with Operate and Cancel. */
export function RealCommandControl({ iedId, iedName, control, options, ready, why }: {
  iedId: string; iedName: string; control: CommandControl; options: CommandOptions; ready: boolean; why: string;
}) {
  const steps = useSboSteps();
  const sbo = isSbo(control.model) && steps;
  const [pending, setPending] = useState<Pending | null>(null);
  const [busy, setBusy] = useState(false);
  const [outcome, setOutcome] = useState<{ ok: boolean; text: string } | null>(null);
  const [problem, setProblem] = useState('');
  const cancelButton = useRef<HTMLButtonElement>(null);
  useEffect(() => { if (pending) cancelButton.current?.focus({ preventScroll: true }); }, [pending]);
  const point = { label: control.ref, unit: '' };
  const words = (value: CommandValue) => commandWords(point, control, value);
  const step = async (action: StepAction, value?: CommandValue, selected = false) => {
    const answer = await sendStep(iedId, { action, ref: control.ref, value, options, confirm: action !== 'cancel', selected });
    if (!answer.ok) throw new Error(answer.summary);
    return answer;
  };
  const flow = useSelectOperate<CommandValue>({
    select: value => step('select', value),
    operate: async value => answerText(await step('operate', value, true)),
    cancel: () => step('cancel'),
  });
  const held = flow.state.phase !== 'idle';
  const off = !ready || busy || held || !!pending;
  const title = !ready ? why : sbo ? SBO_COPY.sboHint : isSbo(control.model) ? SBO_COPY.sboOneClick : SBO_COPY.directHint;
  const verb = sbo ? 'Select' : 'Send';
  const pick = (value: CommandValue) => {
    flow.clear(); setOutcome(null); setProblem('');
    setPending({ action: sbo ? 'select' : 'operate', label: words(value), value });
  };
  const confirm = async () => {
    if (!pending) return;
    if (pending.action === 'select') { setPending(null); void flow.select(pending.value); return; }
    setBusy(true);
    try { setOutcome({ ok: true, text: answerText(await step('operate', pending.value)) }); }
    catch (error) { setOutcome({ ok: false, text: (error as Error).message }); }
    finally { setBusy(false); setPending(null); }
  };
  const input = commandInput(control);
  let body: ReactNode;
  if (input === 'number') body = <NumberBox point={point} control={control} disabled={off} title={title} verb={verb} pick={pick} problem={setProblem} />;
  else if (input === 'list') body = <ListBox point={point} control={control} disabled={off} title={title} verb={verb} pick={pick} />;
  else if (input === 'steps') body = <StepButtons disabled={off} title={title} label={`${verb} ${control.ref}`} press={pick} />;
  else body = <span className="pp-actions" role="group" aria-label={`Send ${control.ref}`}>
    {commandChoices(control).map(choice => <button key={choice.label} type="button" className="wire" disabled={off} title={title}
      onClick={() => pick(choice.value)}><Send size={13} aria-hidden="true" />
      {sbo ? `Select ${choice.label.replace(/^Send /, '')}` : choice.label.replace(/^Send /, '')}</button>)}</span>;
  // Inputs sit beside the reference; the question, countdown and answer take their own full line.
  return <>
    <span className="real-command-inputs">{body}{busy && <span className="small muted">{SBO_COPY.operating}</span>}</span>
    {problem && <span className="small sbo-note real-command-line" role="alert">{problem}</span>}
    {pending && <span className="confirm-box real-confirm real-command-line" role="alertdialog" aria-label="Confirm command">
      <span className="real-confirm-title"><TriangleAlert size={16} aria-hidden="true" />
        <strong>{pending.action === 'select' ? COMMAND_COPY.confirmSelect(pending.label, control.ref, iedName)
          : GATEWAY_COPY.confirmTitle(pending.label, `${iedName} · ${control.ref}`)}</strong></span>
      <span className="small">{pending.action === 'select' ? COMMAND_COPY.confirmSelectHint : GATEWAY_COPY.realWarning}
        {options.test ? ` ${SBO_COPY.test}: on.` : ''}</span>
      <span className="row" style={{ gap: 8 }}>
        <button ref={cancelButton} type="button" onClick={() => setPending(null)}>{GATEWAY_COPY.cancel}</button>
        <button type="button" className="danger wire" onClick={() => { void confirm(); }}><Send size={14} aria-hidden="true" />
          {pending.action === 'select' ? COMMAND_COPY.yesSelect(pending.label) : GATEWAY_COPY.send(pending.label)}</button>
      </span></span>}
    <span className="real-command-line"><SboCountdown flow={flow} what={words} compact /></span>
    {outcome && <span className={`small popup-result real-command-line${outcome.ok ? ' ok' : ''}`} role="status">{outcome.text}</span>}
  </>;
}
