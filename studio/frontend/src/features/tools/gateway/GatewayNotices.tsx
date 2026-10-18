import { Play, TriangleAlert } from 'lucide-react';
import type { GatewayStatus } from '../../../api/gateway';
import { GATEWAY_COPY } from '../../../copy/gateway';
import { HelpTip } from '../../../ui/HelpTip';

/** Run is separate from the gateway: say when the IEDs are off and offer Run, never start it silently.
 *  A compact line in the settings column, the full reason behind "?" (change 0093). */
export function GatewayNotices({ status, runState, runIeds }: {
  status: GatewayStatus; runState: string; runIeds: () => void;
}) {
  const iedsOff = runState === 'stopped';
  if (!iedsOff && status.problems.length === 0) return null;
  return <div className="gateway-notices">
    {iedsOff && <div className="gateway-run-note" role="note">
      <span className="label-row small"><TriangleAlert size={14} aria-hidden /><strong>{GATEWAY_COPY.iedsOff}</strong>
        <HelpTip text={status.state === 'running' ? GATEWAY_COPY.notRunning : GATEWAY_COPY.separateRun} label="Why does this matter?" /></span>
      <button onClick={runIeds}><Play size={15} />{GATEWAY_COPY.runIeds}</button>
    </div>}
    {status.problems.length > 0 && <details className="side-details small gateway-warning">
      <summary>⚠ {GATEWAY_COPY.skipped(status.problems.length)}</summary>
      <ul>{status.problems.slice(0, 5).map(problem =>
        <li key={problem}>{status.role === 'converter' ? problem : `Skipped: ${problem}`}</li>)}</ul></details>}
  </div>;
}
