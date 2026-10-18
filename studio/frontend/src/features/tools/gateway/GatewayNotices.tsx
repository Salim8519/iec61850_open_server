import { Play } from 'lucide-react';
import type { GatewayStatus } from '../../../api/gateway';
import { GATEWAY_COPY } from '../../../copy/gateway';

/** Run is separate from the gateway: say when the IEDs are off and offer Run, never start it silently. */
export function GatewayNotices({ status, runState, runIeds }: {
  status: GatewayStatus; runState: string; runIeds: () => void;
}) {
  const iedsOff = runState === 'stopped';
  if (!iedsOff && status.problems.length === 0) return null;
  return <div className="gateway-notices">
    {iedsOff && <div className="row gateway-run-note" role="note">
      <span className="small">{status.state === 'running' ? GATEWAY_COPY.notRunning : GATEWAY_COPY.separateRun}</span>
      <button onClick={runIeds}><Play size={15} />{GATEWAY_COPY.runIeds}</button>
    </div>}
    {status.problems.length > 0 && <ul className="small gateway-warning">{status.problems.slice(0, 5).map(problem =>
      <li key={problem}>{status.role === 'converter' ? problem : `Skipped: ${problem}`}</li>)}</ul>}
  </div>;
}
