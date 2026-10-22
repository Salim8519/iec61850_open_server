import { Repeat, Square, TriangleAlert } from 'lucide-react';
import { useEffect, useState } from 'react';
import { masterApi, type CyclicState, type MasterCommand } from '../../../../api/master';
import { liveFeed } from '../../../../api/polls';
import { CYCLIC_COPY } from '../../../../copy/master';

const CYCLIC = '/tools/master/cyclic';

/** Repeat one command on a timer, optionally stepping the address and flipping the value.
 *  A real-device action, so Start is behind the same warning as a single command. */
export function MasterCyclic({ command, ready, fail }: {
  command: MasterCommand; ready: boolean; fail: (message: string) => void;
}) {
  const [state, setState] = useState<CyclicState | null>(null);
  const [interval, setInterval] = useState(1000);
  const [step, setStep] = useState(false);
  const [toggle, setToggle] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  useEffect(() => liveFeed.subscribe<CyclicState>(CYCLIC, 1000, setState, reason => fail(reason.message)), [fail]);
  const running = state?.running ?? false;
  const start = () => {
    setBusy(true);
    masterApi.startCyclic(command, { interval_ms: interval, address_step: step ? 1 : 0, toggle })
      .then(setState).catch(error => fail(error.message)).finally(() => { setBusy(false); setConfirming(false); });
  };
  const stop = () => {
    setBusy(true);
    masterApi.stopCyclic().then(setState).catch(error => fail(error.message)).finally(() => setBusy(false));
  };
  return <div className="master-cyclic" role="group" aria-label={CYCLIC_COPY.title}>
    <div className="row"><Repeat size={14} aria-hidden /><strong>{CYCLIC_COPY.title}</strong></div>
    <p className="small muted" style={{ margin: 0 }}>{CYCLIC_COPY.hint}</p>
    <div className="row master-command-fields">
      <label>{CYCLIC_COPY.every}<input inputMode="numeric" className="gateway-number" value={interval}
        disabled={running} onChange={event => setInterval(Number(event.target.value) || 0)} /></label>
      <label className="row gateway-check"><input type="checkbox" checked={step} disabled={running}
        onChange={event => setStep(event.target.checked)} />{CYCLIC_COPY.step}</label>
      <label className="row gateway-check"><input type="checkbox" checked={toggle} disabled={running}
        onChange={event => setToggle(event.target.checked)} />{CYCLIC_COPY.toggle}</label>
      {running
        ? <button className="danger" disabled={busy} onClick={stop}><Square size={14} />{CYCLIC_COPY.stop}</button>
        : <button className="primary" disabled={!ready || confirming} onClick={() => setConfirming(true)}>
          <Repeat size={14} />{CYCLIC_COPY.start}</button>}
    </div>
    {running && <p className="small gateway-sent ok" role="status">{CYCLIC_COPY.running(state?.sends ?? 0)}</p>}
    {confirming && !running && <div className="confirm-box gateway-confirm real" role="alertdialog" aria-label="Confirm repeat">
      <p><TriangleAlert size={18} aria-hidden /><strong>{CYCLIC_COPY.warn}</strong></p>
      <div className="row"><button onClick={() => setConfirming(false)} autoFocus>Cancel</button>
        <button className="danger" disabled={busy} onClick={start}><Repeat size={14} />{CYCLIC_COPY.start}</button></div>
    </div>}
  </div>;
}
