import { Send, TriangleAlert, X } from 'lucide-react';
import { useState } from 'react';
import { masterApi, type CommandKind, type CrobCode, type MasterCommand, type MasterProtocol } from '../../../../api/master';
import { ACTIVATION, ADDRESS_WORD, COMMAND_HINT, COMMAND_NAMES, COMMANDS, CROB_CODES, MASTER_COPY, REG_VALUES, TARGET_WORD } from '../../../../copy/master';
import { MasterCyclic } from './MasterCyclic';
import '../../../../ui/kind.css';

const ON_OFF = [{ value: 1, label: 'ON (1)' }, { value: 0, label: 'OFF (0)' }];
const CHOICES: Partial<Record<CommandKind, { value: number; label: string }[]>> = {
  double: [{ value: 2, label: MASTER_COPY.doubleValues[2] }, { value: 1, label: MASTER_COPY.doubleValues[1] }],
  single: ON_OFF, binary_output: ON_OFF, coil: ON_OFF,  // a coil is ON or OFF, never a number (change 0089)
  regulating: REG_VALUES,
};
const TRIP_CLOSE = [{ value: 1, label: 'Close' }, { value: 0, label: 'Trip' }];

/** A sensible starting value for a command type. */
export const startValue = (kind: CommandKind) =>
  kind === 'double' || kind === 'regulating' ? 2 : kind === 'single' || kind === 'binary_output' || kind === 'coil' ? 1 : 0;

/** Build one command, then confirm it against the strong REAL-device warning before it leaves.
 *  `review` opens straight at that confirmation (a list point's Send button, ADR-0110): one click, then Yes. */
export function MasterCommandForm({ draft, host, protocol, close, fail, review = false }: {
  draft: MasterCommand; host: string; protocol: MasterProtocol; close: () => void; fail: (message: string) => void;
  review?: boolean;
}) {
  const [command, setCommand] = useState<MasterCommand>(draft);
  const [confirming, setConfirming] = useState(review);
  const [sending, setSending] = useState(false);
  const [answer, setAnswer] = useState<{ ok: boolean; text: string } | null>(null);
  const clock = command.kind === 'clock';
  const crob = command.kind === 'binary_output';
  const iec104 = protocol === 'iec104';
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
  const numberField = (label: string, key: 'qualifier' | 'count' | 'on_time_ms' | 'off_time_ms', fallback: number, hint?: string) =>
    <label title={hint}>{label}<input inputMode="numeric" className="gateway-number" defaultValue={String(command[key] ?? fallback)}
      onBlur={event => change({ [key]: event.target.value === '' ? undefined : Number(event.target.value) })} /></label>;
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
      {!clock && <label>{MASTER_COPY.value}{valueField}</label>}
      {iec104 && !clock && numberField(MASTER_COPY.qualifier, 'qualifier', 0, MASTER_COPY.qualifierHint)}
      {iec104 && !clock && <label title={MASTER_COPY.activationHint}>{MASTER_COPY.activation}
        <select value={command.deactivate ? 1 : 0} onChange={event => change({ deactivate: Number(event.target.value) === 1 })}>
          {ACTIVATION.map(choice => <option key={choice.value} value={choice.value}>{choice.label}</option>)}</select></label>}
      {!clock && !command.deactivate && protocol !== 'modbus' && <label className="row gateway-check">
        <input type="checkbox" checked={Boolean(command.select)} onChange={event => change({ select: event.target.checked })} />
        {MASTER_COPY.select}</label>}
      <button className="primary" disabled={!ready || confirming} onClick={() => setConfirming(true)}>
        <Send size={14} />Review…</button>
    </div>
    {crob && <div className="row master-command-fields" title={MASTER_COPY.crobTimingHint}>
      {numberField(MASTER_COPY.crobCount, 'count', 1)}{numberField(MASTER_COPY.crobOn, 'on_time_ms', 100)}
      {numberField(MASTER_COPY.crobOff, 'off_time_ms', 100)}</div>}
    {!clock && <p className="small muted" style={{ margin: 0 }}>{COMMAND_HINT[protocol]}</p>}
    {confirming && <div className="confirm-box gateway-confirm real" role="alertdialog" aria-label="Confirm command">
      <p><TriangleAlert size={18} aria-hidden /><strong>{MASTER_COPY.confirmTitle(COMMAND_NAMES[command.kind],
        clock ? undefined : command.address, host, TARGET_WORD[protocol])}</strong></p>
      <p className="small">{MASTER_COPY.realWarning}</p>
      <div className="row"><button onClick={() => setConfirming(false)} autoFocus>{MASTER_COPY.cancel}</button>
        <button className="danger" disabled={sending} onClick={send}><Send size={14} />{MASTER_COPY.send}</button></div>
    </div>}
    {answer && <p className={`small gateway-sent${answer.ok ? ' ok' : ''}`} role="status">
      {answer.ok ? 'Done: ' : 'Not carried out: '}{answer.text}</p>}
    {!clock && <MasterCyclic command={command} ready={ready} fail={fail} />}
  </div>;
}
