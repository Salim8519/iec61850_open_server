import { Send, TriangleAlert, X } from 'lucide-react';
import { useState } from 'react';
import { masterApi, type CommandKind, type CrobCode, type MasterCommand, type MasterProtocol } from '../../../../api/master';
import { ADDRESS_WORD, COMMAND_HINT, COMMAND_NAMES, COMMANDS, CROB_CODES, MASTER_COPY, TARGET_WORD } from '../../../../copy/master';

const ON_OFF = [{ value: 1, label: 'ON (1)' }, { value: 0, label: 'OFF (0)' }];
const CHOICES: Partial<Record<CommandKind, { value: number; label: string }[]>> = {
  double: [{ value: 2, label: MASTER_COPY.doubleValues[2] }, { value: 1, label: MASTER_COPY.doubleValues[1] }],
  single: ON_OFF, binary_output: ON_OFF, coil: ON_OFF,  // a coil is ON or OFF, never a number (change 0089)
};
const TRIP_CLOSE = [{ value: 1, label: 'Close' }, { value: 0, label: 'Trip' }];

/** A sensible starting value for a command type. */
export const startValue = (kind: CommandKind) => kind === 'double' ? 2 : kind === 'single' || kind === 'binary_output' || kind === 'coil' ? 1 : 0;

/** Build one command, then confirm it against the strong REAL-device warning before it leaves. */
export function MasterCommandForm({ draft, host, protocol, close, fail }: {
  draft: MasterCommand; host: string; protocol: MasterProtocol; close: () => void; fail: (message: string) => void;
}) {
  const [command, setCommand] = useState<MasterCommand>(draft);
  const [confirming, setConfirming] = useState(false);
  const [sending, setSending] = useState(false);
  const [answer, setAnswer] = useState<{ ok: boolean; text: string } | null>(null);
  const clock = command.kind === 'clock';
  const crob = command.kind === 'binary_output';
  const ready = clock || (command.address !== undefined && Number.isInteger(command.address) && Number.isFinite(command.value));
  const change = (patch: Partial<MasterCommand>) => { setCommand({ ...command, ...patch }); setConfirming(false); setAnswer(null); };
  const send = () => {
    setSending(true);
    masterApi.command(clock ? { kind: 'clock' } : command).then(setAnswer).catch(error => fail(error.message))
      .finally(() => { setSending(false); setConfirming(false); });
  };
  const choices = crob && command.code === 'trip_close' ? TRIP_CLOSE : CHOICES[command.kind];
  const valueField = choices
    ? <select value={command.value} aria-label={MASTER_COPY.value} onChange={event => change({ value: Number(event.target.value) })}>
      {choices.map(choice => <option key={choice.value} value={choice.value}>{choice.label}</option>)}</select>
    : <input inputMode="decimal" defaultValue={String(command.value ?? 0)} aria-label={MASTER_COPY.value}
      onBlur={event => change({ value: Number(event.target.value) })} />;
  return <div className="master-command" role="group" aria-label={MASTER_COPY.commandTitle}>
    <div className="row gateway-toolbar"><strong>{MASTER_COPY.commandTitle}</strong><span className="gateway-spacer" />
      <button className="ghost icon-button" onClick={close} aria-label="Close"><X size={16} /></button></div>
    <div className="row master-command-fields">
      <label>{MASTER_COPY.kind}<select value={command.kind} onChange={event => {
        const kind = event.target.value as CommandKind;
        change({ kind, value: startValue(kind), code: kind === 'binary_output' ? command.code ?? 'latch' : undefined });
      }}>{COMMANDS[protocol].map(kind => <option key={kind} value={kind}>{COMMAND_NAMES[kind]}</option>)}</select></label>
      {crob && <label>{MASTER_COPY.code}<select value={command.code ?? 'latch'}
        onChange={event => change({ code: event.target.value as CrobCode, value: 1 })}>
        {(Object.keys(CROB_CODES) as CrobCode[]).map(code => <option key={code} value={code}>{CROB_CODES[code]}</option>)}
      </select></label>}
      {!clock && <label>{ADDRESS_WORD[protocol]}<input inputMode="numeric" className="gateway-number"
        defaultValue={command.address ?? ''} onBlur={event => change({ address: event.target.value === '' ? undefined
          : Number(event.target.value) })} /></label>}
      {!clock && !(crob && command.code === 'pulse') && <label>{MASTER_COPY.value}{valueField}</label>}
      {!clock && protocol !== 'modbus' && <label className="row gateway-check"><input type="checkbox" checked={Boolean(command.select)}
        onChange={event => change({ select: event.target.checked })} />{MASTER_COPY.select}</label>}
      <button className="primary" disabled={!ready || confirming} onClick={() => setConfirming(true)}>
        <Send size={14} />Review…</button>
    </div>
    {!clock && <p className="small muted" style={{ margin: 0 }}>{COMMAND_HINT[protocol]}</p>}
    {confirming && <div className="gateway-confirm real" role="alertdialog" aria-label="Confirm command">
      <p><TriangleAlert size={18} aria-hidden /><strong>{MASTER_COPY.confirmTitle(COMMAND_NAMES[command.kind],
        clock ? undefined : command.address, host, TARGET_WORD[protocol])}</strong></p>
      <p className="small">{MASTER_COPY.realWarning}</p>
      <div className="row"><button onClick={() => setConfirming(false)} autoFocus>{MASTER_COPY.cancel}</button>
        <button className="danger" disabled={sending} onClick={send}><Send size={14} />{MASTER_COPY.send}</button></div>
    </div>}
    {answer && <p className={`small gateway-sent${answer.ok ? ' ok' : ''}`} role="status">
      {answer.ok ? 'Done: ' : 'Not carried out: '}{answer.text}</p>}
  </div>;
}
