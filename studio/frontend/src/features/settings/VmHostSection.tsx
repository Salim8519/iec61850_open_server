import { useCallback, useEffect, useState } from 'react';
import { ServerCog } from 'lucide-react';
import { getVmHost, setVmHost, type VmHostStatus } from '../../api/vmHost';
import { VM_HOST_COPY as COPY, vmHostSummary } from '../../copy/vmHost';
import { HelpTip } from '../../ui/HelpTip';
import { StatusPill } from '../../ui/StatusPill';
import { VmHostFirewall } from './VmHostFirewall';
import './vm-host.css';

/**
 * Settings › License › "Let my virtual machines use this license" (ADR-0099). Only a PC with a lifetime
 * license shows it. Turning it on starts a small helper that answers the PC's own virtual machines;
 * turning it off ends the helper. Hidden when the backend has no such routes (the frozen Python engine).
 */
export function VmHostSection() {
  const [status, setStatus] = useState<VmHostStatus | null>(null);
  const [hidden, setHidden] = useState(false);
  const [problem, setProblem] = useState('');
  const [busy, setBusy] = useState(false);
  const load = useCallback(() => {
    getVmHost().then(setStatus).catch(() => setHidden(true));
  }, []);
  useEffect(load, [load]);
  if (hidden) return null;
  if (!status) return <div className="vm-host"><p className="small muted" style={{ margin: 0 }}>{COPY.reading}</p></div>;
  const change = (enabled: boolean) => {
    setBusy(true); setProblem('');
    setVmHost(enabled).then(setStatus).catch(error => setProblem(`${COPY.turnOnFailed}: ${error.message}`))
      .finally(() => setBusy(false));
  };
  const blocked = !status.available && !status.enabled;
  const summary = vmHostSummary(status.enabled, status.running, status.helper_found);
  return <div className="vm-host" role="group" aria-label={COPY.title}>
    <h3 className="label-row vm-host-title"><ServerCog size={16} aria-hidden="true" />{COPY.title}
      <HelpTip text={COPY.how} label="How does this work?" /></h3>
    <p className="small muted" style={{ margin: 0 }}>{COPY.intro}</p>
    <label className="row small vm-host-switch">
      <input type="checkbox" checked={status.enabled} disabled={busy || blocked}
        onChange={event => change(event.target.checked)} />
      <span><strong>{COPY.switchLabel}</strong></span>
    </label>
    {blocked && status.reason && <p className="small vm-host-reason" role="status">{status.reason}</p>}
    {!blocked && <p className="small muted vm-host-state" style={{ margin: 0 }}>
      <StatusPill status={{ label: status.enabled ? 'On' : 'Off', tone: status.enabled ? 'good' : 'off', hint: summary }} />
      {' '}{summary}</p>}
    {status.available && status.key && <p className="small muted vm-host-key" style={{ margin: 0 }}>
      {COPY.keyLabel}: <code>{status.key}</code> <HelpTip text={COPY.keyHint} label="What is this key?" /></p>}
    {status.enabled && status.available && <VmHostFirewall port={status.port} />}
    {problem && <p className="small vm-host-reason" role="alert">{problem}</p>}
  </div>;
}
