import { ShieldAlert, ShieldCheck } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import type { ApplyResult } from '../../api/network';
import { applyVmFirewall, getVmFirewall, type VmFirewallPreview } from '../../api/vmHost';
import { VM_HOST_COPY as COPY } from '../../copy/vmHost';
import { HelpTip } from '../../ui/HelpTip';

/**
 * The one Windows Firewall rule the virtual machine helper needs, reviewed before it is applied and
 * made only by the network helper (journaled, removed by Settings › Network › Clean up). Same flow as
 * the gateway rule: show what changes, ask, then apply what was reviewed.
 */
export function VmHostFirewall({ port }: { port: number }) {
  const [preview, setPreview] = useState<VmFirewallPreview | null>(null);
  const [problem, setProblem] = useState('');
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<ApplyResult | null>(null);
  const check = useCallback(() => {
    setProblem('');
    getVmFirewall().then(setPreview).catch(error => setProblem(`${COPY.firewallFailed}: ${error.message}`));
  }, []);
  useEffect(check, [check, port]);
  const apply = () => {
    setBusy(true);
    applyVmFirewall(preview!.fingerprint).then(answer => { setResult(answer); setConfirming(false); check(); })
      .catch(error => setProblem(error.message)).finally(() => setBusy(false));
  };
  if (problem) return <p className="small vm-host-reason" role="alert" style={{ margin: 0 }}>{problem}</p>;
  if (!preview) return <p className="small muted" style={{ margin: 0 }}>{COPY.firewallChecking}</p>;
  if (preview.open) return <p className="small row" style={{ margin: 0 }}>
    <ShieldCheck size={16} aria-hidden="true" /><span>✓ {COPY.firewallOpen}{result ? ` ${result.message}` : ''}</span></p>;
  return <div className="vm-host-firewall" role="group" aria-label={COPY.firewallTitle}>
    <p className="small vm-host-reason label-row" style={{ margin: 0 }}><ShieldAlert size={16} aria-hidden="true" />
      <strong>⚠ {COPY.firewallClosed}</strong><HelpTip text={COPY.firewallWhy} label="Why?" /></p>
    <details className="side-details small"><summary>{COPY.firewallChanges}</summary><ul>
      {preview.items.filter(item => item.action === 'add').map(item => <li key={item.description}>{item.description}</li>)}
    </ul></details>
    {result && !result.applied && <ul className="small vm-host-reason">
      {result.results.filter(entry => !entry.ok).map(entry => <li key={entry.description}>{entry.description}: {entry.message}</li>)}</ul>}
    {!confirming ? <button onClick={() => setConfirming(true)}>{COPY.firewallReview}</button>
      : <div className="vm-host-confirm"><span className="small">{COPY.firewallAdmin}</span>
        <div className="row">
          <button className="primary" disabled={busy} onClick={apply}>{COPY.firewallApply}</button>
          <button className="ghost" disabled={busy} onClick={() => setConfirming(false)}>{COPY.firewallCancel}</button></div></div>}
  </div>;
}
