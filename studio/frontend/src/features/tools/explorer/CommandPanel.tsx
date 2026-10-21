import { Loader2, Send } from 'lucide-react';
import { useState } from 'react';
import { explorerApi, type Attribute, type OperateAnswer } from '../../../api/tools';
import { EXPLORER_COPY } from '../../../copy/tools';
import '../../../ui/kind.css';
import { commandChoices, type Choice } from './commandChoices';
import { ValueText } from './ValueText';

/** Send a command to the connected device, always after one small confirmation. */
export function CommandPanel({ reference, host, attributes, fail, operated }: {
  reference: string; host: string; attributes: Attribute[]; fail: (message: string) => void; operated?: () => void;
}) {
  const choices = commandChoices(attributes);
  const [pending, setPending] = useState<Choice | null>(null);
  const [step, setStep] = useState('1');
  const [busy, setBusy] = useState(false);
  const [answer, setAnswer] = useState<OperateAnswer | null>(null);
  if (!choices) return null;
  const send = async (choice: Choice) => {
    setBusy(true); setAnswer(null);
    try { setAnswer(await explorerApi.operate(reference, choice.value)); }
    catch (error) { fail((error as Error).message); }
    finally { setBusy(false); setPending(null); operated?.(); }  // the table below must not keep the old value (change 0089)
  };
  const options: Choice[] = choices === 'number' ? [{ label: `Send ${step}`, value: Number(step) }] : choices;
  return <div className="command-panel">
    <div className="row command-row">
      <span className="small muted">{EXPLORER_COPY.command}</span>
      {choices === 'number' && <input className="command-step" type="number" value={step} aria-label="Value to send"
        onChange={event => setStep(event.target.value)} />}
      {!pending && options.map(choice => <button key={choice.label} disabled={busy || !Number.isFinite(Number(choice.value))}
        onClick={() => { setAnswer(null); setPending(choice); }}><Send size={14} />{choice.label}…</button>)}
      {busy && <Loader2 size={16} className="spin" />}
    </div>
    {pending && <div className="confirm-box command-confirm" role="alertdialog" aria-label="Confirm command">
      <p><strong>{EXPLORER_COPY.confirmTitle(pending.label, reference, host)}</strong></p>
      <p className="small">{EXPLORER_COPY.confirmHint}</p>
      <div className="row">
        <button onClick={() => setPending(null)} autoFocus>{EXPLORER_COPY.cancel}</button>
        <button className="danger" disabled={busy} onClick={() => send(pending)}>
          <Send size={14} />{EXPLORER_COPY.send(pending.label)}</button>
      </div>
    </div>}
    {answer && <div className="command-result small" role="status">
      <ValueText text={answer.summary} status={answer.result.ok ? 'ok' : 'error'} />
      {answer.result.add_cause_text && <span className="muted"> · {answer.result.add_cause_text}</span>}
      {answer.observed && <div>{EXPLORER_COPY.nowReads} <ValueText text={answer.observed.text} status={answer.observed.status} /></div>}
    </div>}
  </div>;
}
