import { Send, TriangleAlert } from 'lucide-react';
import type { GatewayPoint } from '../../../api/gateway';
import { GATEWAY_COPY } from '../../../copy/gateway';
import { commandLabels } from './gatewayCommands';

export type PendingCommand = { row: GatewayPoint; value: boolean; label: string };

/** Send buttons in a command row; they only open the warning, they never send by themselves. */
export function CommandButtons({ row, blocked, ask }: {
  row: GatewayPoint; blocked: string; ask: (pending: PendingCommand) => void;
}) {
  const [on, off] = commandLabels(row);
  if (blocked) return <span className="small muted">{blocked}</span>;
  return <span className="row gateway-send">
    <button onClick={() => ask({ row, value: true, label: on })}><Send size={14} />{on}…</button>
    <button onClick={() => ask({ row, value: false, label: off })}><Send size={14} />{off}…</button>
  </span>;
}

/** The warning shown before a command leaves the page. Real devices get the strongest wording. */
export function CommandWarning({ pending, real, busy, cancel, send }: {
  pending: PendingCommand; real: boolean; busy: boolean; cancel: () => void; send: () => void;
}) {
  const target = `${pending.row.ied_name ?? pending.row.ied} · ${pending.row.ref}`;
  return <div className={`gateway-confirm${real ? ' real' : ''}`} role="alertdialog" aria-label="Confirm command">
    <p><TriangleAlert size={18} aria-hidden /><strong>{GATEWAY_COPY.confirmTitle(pending.label, target)}</strong></p>
    <p className="small">{real ? GATEWAY_COPY.realWarning : GATEWAY_COPY.simulatedWarning}</p>
    <div className="row">
      <button onClick={cancel} autoFocus>{GATEWAY_COPY.cancel}</button>
      <button className="danger" disabled={busy} onClick={send}><Send size={14} />{GATEWAY_COPY.send(pending.label)}</button>
    </div>
  </div>;
}
