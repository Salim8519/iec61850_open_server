import { Send, TriangleAlert } from 'lucide-react';
import { useState } from 'react';
import type { CommandValue, GatewayPoint } from '../../../api/gateway';
import { GATEWAY_COPY, OWN_COPY } from '../../../copy/gateway';
import { TYPED_COMMAND_COPY } from '../../../copy/gatewayTypedKinds';
import '../../../ui/kind.css';
import { commandLabels, commandValues, parseSetpoint } from './gatewayCommands';
import { ownName } from './ownPoints';

export type PendingCommand = { row: GatewayPoint; value: CommandValue; label: string };

/** Send buttons in a command row; they only open the warning, they never send by themselves. A set
 *  point row has a number box and one Send button (ADR-0107). */
export function CommandButtons({ row, blocked, ask }: {
  row: GatewayPoint; blocked: string; ask: (pending: PendingCommand) => void;
}) {
  if (blocked) return <span className="small muted">{blocked}</span>;
  if (row.kind === 'command_setpoint') return <SetpointSend row={row} ask={ask} />;
  const [on, off] = commandLabels(row);
  const [first, second] = commandValues(row);
  return <span className="row gateway-send">
    <button onClick={() => ask({ row, value: first, label: on })}><Send size={14} />{on}…</button>
    <button onClick={() => ask({ row, value: second, label: off })}><Send size={14} />{off}…</button>
  </span>;
}

function SetpointSend({ row, ask }: { row: GatewayPoint; ask: (pending: PendingCommand) => void }) {
  const [text, setText] = useState('');
  const [problem, setProblem] = useState('');
  const send = () => {
    const parsed = parseSetpoint(text);
    if ('error' in parsed) { setProblem(parsed.error); return; }
    setProblem('');
    ask({ row, value: parsed.value, label: TYPED_COMMAND_COPY.setpointLabel(parsed.value) });
  };
  return <span className="row gateway-send">
    <input className="gateway-number" inputMode="decimal" value={text} placeholder="0" aria-label={TYPED_COMMAND_COPY.value}
      title={problem || OWN_COPY.scaleHint} aria-invalid={problem ? true : undefined}
      onChange={event => { setText(event.target.value); setProblem(''); }}
      onKeyDown={event => { if (event.key === 'Enter') send(); }} />
    <button onClick={send}><Send size={14} />{TYPED_COMMAND_COPY.send}…</button>
    {problem && <span className="small gateway-warning" role="alert">{problem}</span>}
  </span>;
}

/** The warning shown before a command leaves the page. Real devices get the strongest wording. */
export function CommandWarning({ pending, real, own, busy, cancel, send }: {
  pending: PendingCommand; real: boolean; own?: boolean; busy: boolean; cancel: () => void; send: () => void;
}) {
  const target = own ? `${OWN_COPY.ownPoint} · ${ownName(pending.row)}` : `${pending.row.ied_name ?? pending.row.ied} · ${pending.row.ref}`;
  return <div className={`confirm-box gateway-confirm${real ? ' real' : ''}`} role="alertdialog" aria-label="Confirm command">
    <p><TriangleAlert size={18} aria-hidden /><strong>{GATEWAY_COPY.confirmTitle(pending.label, target)}</strong></p>
    <p className="small">{own ? OWN_COPY.commandWarning : real ? GATEWAY_COPY.realWarning : GATEWAY_COPY.simulatedWarning}</p>
    <div className="row">
      <button onClick={cancel} autoFocus>{GATEWAY_COPY.cancel}</button>
      <button className="danger" disabled={busy} onClick={send}><Send size={14} />{GATEWAY_COPY.send(pending.label)}</button>
    </div>
  </div>;
}
