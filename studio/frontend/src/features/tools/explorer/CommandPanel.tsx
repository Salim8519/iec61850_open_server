import { ArrowDown, ArrowUp, Loader2, Send, X } from 'lucide-react';
import { useState } from 'react';
import { explorerApi, type Attribute, type CommandAnswer, type CommandOptions, type ReadValue } from '../../../api/tools';
import { COMMAND_COPY } from '../../../copy/commands';
import { EXPLORER_COPY } from '../../../copy/tools';
import '../../../ui/kind.css';
import { CommandOptionsBox, DEFAULT_OPTIONS } from './CommandOptionsBox';
import { commandSpec, modelText, needsSelect, parsedNumber } from './commandSpec';
import { useSelection } from './useSelection';
import { ValueText } from './ValueText';

type Pending = { action: 'select' | 'operate'; label: string; value: unknown };

/** Send a typed command to the connected device (ADR-0104), always after one small confirmation. Controls
 *  that select before operate show Select, then a countdown with Operate and Cancel. */
export function CommandPanel({ reference, name, host, attributes, values, fail, operated }: {
  reference: string; name: string; host: string; attributes: Attribute[]; values: Record<string, ReadValue>;
  fail: (message: string) => void; operated?: () => void;
}) {
  const spec = commandSpec(name, attributes, values);
  const [text, setText] = useState('1');
  const [listed, setListed] = useState(1);
  const [options, setOptions] = useState<CommandOptions>(DEFAULT_OPTIONS);
  const [pending, setPending] = useState<Pending | null>(null);
  const [busy, setBusy] = useState(false);
  const [answer, setAnswer] = useState<CommandAnswer | null>(null);
  const [notice, setNotice] = useState('');
  // A timed-out selection also drops the old "selected" answer, so the two never contradict (change 0177).
  const select = useSelection(() => { setNotice(COMMAND_COPY.expired); setAnswer(null); });
  if (!spec) return null;
  const sbo = needsSelect(spec.model);
  const run = async (action: 'select' | 'operate' | 'cancel', label: string, value: unknown) => {
    setBusy(true); setAnswer(null); setNotice('');
    try {
      const reply = await explorerApi.command(action, reference, spec.kind, value, options);
      setAnswer(reply);
      if (action === 'select' && reply.result.ok && reply.result.selected) select.start(label, value, reply.result.expires_ms ?? 0);
      if (action !== 'select') select.clear();
    } catch (error) { fail((error as Error).message); }
    finally { setBusy(false); setPending(null); if (action === 'operate') operated?.(); }  // read again (change 0089)
  };
  const send = (item: Pending) => run(item.action, item.label, item.value);
  const ask = (label: string, value: unknown) => { setAnswer(null); setPending({ action: sbo ? 'select' : 'operate', label, value }); };
  const word = (label: string) => sbo ? COMMAND_COPY.selectFirst(label) : COMMAND_COPY.sendNow(label);
  const number = parsedNumber(spec.kind, text);
  const enumLabel = spec.labels.find(item => item.value === listed)?.label ?? String(listed);
  const idle = !pending && !select.selection;
  return <div className="command-panel">
    <div className="row command-row">
      <span className="small muted">{EXPLORER_COPY.command}</span>
      {idle && spec.kind === 'bool' && spec.choices.map(choice => <button key={choice.label} disabled={busy}
        onClick={() => ask(choice.label, choice.value)}><Send size={14} />{word(choice.label)}</button>)}
      {idle && spec.kind === 'enum' && <>
        <select className="command-pick" value={listed} aria-label={COMMAND_COPY.pickLabel}
          onChange={event => setListed(Number(event.target.value))}>
          {spec.labels.map(item => <option key={item.value} value={item.value}>{item.label}</option>)}</select>
        <button disabled={busy} onClick={() => ask(enumLabel, listed)}><Send size={14} />{word(enumLabel)}</button></>}
      {idle && (spec.kind === 'int' || spec.kind === 'float') && <>
        <input className="command-step" type="number" step={spec.kind === 'int' ? 1 : 'any'} value={text}
          aria-label={COMMAND_COPY.valueLabel} onChange={event => setText(event.target.value)} />
        <button disabled={busy || number === null} onClick={() => ask(String(number), number)}>
          <Send size={14} />{word(String(number ?? text))}</button></>}
      {idle && spec.kind === 'step' && <>
        <button disabled={busy} onClick={() => ask(COMMAND_COPY.higher, 'higher')}><ArrowUp size={14} />{word(COMMAND_COPY.higher)}</button>
        <button disabled={busy} onClick={() => ask(COMMAND_COPY.lower, 'lower')}><ArrowDown size={14} />{word(COMMAND_COPY.lower)}</button></>}
      {busy && <Loader2 size={16} className="spin" />}
    </div>
    <p className="small muted command-model">{COMMAND_COPY.model(modelText(spec.model))}
      {spec.min !== null && spec.max !== null && spec.max > spec.min && <> · {COMMAND_COPY.limits(spec.min, spec.max)}</>}</p>
    {select.selection && <div className="command-selected" role="status">
      <strong>{COMMAND_COPY.selected(select.selection.label, select.left)}</strong>
      <div className="row">
        <button className="danger" disabled={busy} onClick={() => run('operate', select.selection!.label, select.selection!.value)}>
          <Send size={14} />{COMMAND_COPY.operate}</button>
        <button disabled={busy} onClick={() => run('cancel', select.selection!.label, null)}><X size={14} />{COMMAND_COPY.cancelSelect}</button>
      </div></div>}
    {pending && <div className="confirm-box command-confirm" role="alertdialog" aria-label="Confirm command">
      <p><strong>{pending.action === 'select' ? COMMAND_COPY.confirmSelect(pending.label, reference, host)
        : EXPLORER_COPY.confirmTitle(pending.label, reference, host)}</strong></p>
      <p className="small">{pending.action === 'select' ? COMMAND_COPY.confirmSelectHint : EXPLORER_COPY.confirmHint}</p>
      <div className="row">
        <button onClick={() => setPending(null)} autoFocus>{EXPLORER_COPY.cancel}</button>
        <button className="danger" disabled={busy} onClick={() => send(pending)}>
          <Send size={14} />{pending.action === 'select' ? COMMAND_COPY.yesSelect(pending.label) : EXPLORER_COPY.send(pending.label)}</button>
      </div>
    </div>}
    {notice && <p className="small command-notice" role="status">{notice}</p>}
    {answer && <div className="command-result small" role="status">
      <ValueText text={answer.summary} status={answer.result.ok ? 'ok' : 'error'} />
      {answer.result.add_cause_text && <span className="muted"> · {answer.result.add_cause_text}</span>}
    </div>}
    {idle && <CommandOptionsBox options={options} change={setOptions} />}
  </div>;
}
