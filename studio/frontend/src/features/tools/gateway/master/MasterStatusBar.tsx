import { RefreshCw, Settings2, Unplug } from 'lucide-react';
import type { MasterStatus } from '../../../../api/master';
import { PROTOCOL_NAMES } from '../../../../copy/gateway';
import { ASK_AGAIN, MASTER_COPY, RANGES_COPY, SIDES } from '../../../../copy/master';
import { StatusPill } from '../../../../ui/StatusPill';

/** One compact line while connected, so the discovered points get the rest of the screen. */
export function MasterStatusBar({ status, busy, disconnect, showSettings, interrogate }: {
  status: MasterStatus; busy: boolean; disconnect: () => void; showSettings: () => void; interrogate: () => void;
}) {
  const good = status.state === 'connected';
  return <div className="card gateway-bar" role="status">
    <StatusPill status={{ label: MASTER_COPY.states[status.state], tone: good ? 'good' : 'problem', hint: status.message }} />
    <strong>{PROTOCOL_NAMES[status.protocol]} · {SIDES[status.protocol].usShort}</strong>
    <span>{status.host}:{status.port}</span>
    <span className="muted">{MASTER_COPY.points(status.points)}</span>
    {!good && <span className="gateway-warning small">{status.message}</span>}
    <span className="gateway-spacer" />
    {status.poll_failures.length > 0 && <span className="gateway-warning small">{RANGES_COPY.refused(status.poll_failures.length)}</span>}
    <button onClick={interrogate} disabled={busy || !good} title={ASK_AGAIN[status.protocol].hint}>
      <RefreshCw size={16} />{ASK_AGAIN[status.protocol].label}</button>
    <button className="ghost" onClick={showSettings}><Settings2 size={16} />Settings</button>
    <button onClick={disconnect} disabled={busy}><Unplug size={16} />{MASTER_COPY.disconnect}</button>
  </div>;
}
