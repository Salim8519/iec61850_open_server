import { Settings2, Square } from 'lucide-react';
import type { GatewayStatus } from '../../../api/gateway';
import { GATEWAY_COPY, PROTOCOL_NAMES } from '../../../copy/gateway';
import { StatusPill } from '../../../ui/StatusPill';

/** One compact line while the gateway runs, so the point list gets the rest of the screen. */
export function GatewayStatusBar({ status, busy, stop, showSettings }: {
  status: GatewayStatus; busy: boolean; stop: () => void; showSettings: () => void;
}) {
  return <div className="card gateway-bar" role="status">
    <StatusPill status={{ label: 'Running', tone: 'good', hint: status.message }} />
    <strong>{PROTOCOL_NAMES[status.protocol]}</strong>
    <span>{status.message}</span>
    <span className="muted">{status.clients.length
      ? `${GATEWAY_COPY.connected(status.clients.length)}: ${status.clients.join(', ')}` : GATEWAY_COPY.none}</span>
    <span className="gateway-spacer" />
    <button className="ghost" onClick={showSettings}><Settings2 size={16} />{GATEWAY_COPY.showSettings}</button>
    <button onClick={stop} disabled={busy}><Square size={16} />{GATEWAY_COPY.stop}</button>
  </div>;
}
