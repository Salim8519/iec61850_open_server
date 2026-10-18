import { ShieldAlert, ShieldCheck } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../../../api/client';
import type { ApplyResult } from '../../../api/network';
import { FIREWALL_COPY as COPY } from '../../../copy/gateway';
import { HelpTip } from '../../../ui/HelpTip';

type FirewallItem = { kind: string; action: 'add' | 'keep'; description: string; ports?: number[] };
type FirewallPreview = { items: FirewallItem[]; fingerprint: string; open: boolean; blocked: boolean };

/** Shown on "All network adapters": can another computer reach the gateway port, and a
 *  reviewed, one-click fix through the network helper when Windows Firewall stops it (ADR-0062). */
export function GatewayFirewall({ port }: { port: number }) {
  const [preview, setPreview] = useState<FirewallPreview | null>(null);
  const [problem, setProblem] = useState('');
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<ApplyResult | null>(null);
  const check = useCallback(() => {
    setProblem('');
    api<FirewallPreview>('/network/gateway-firewall').then(setPreview)
      .catch(error => setProblem(`${COPY.failed}: ${error.message}`));
  }, []);
  useEffect(check, [check, port]);
  const apply = () => {
    setBusy(true);
    api<ApplyResult>('/network/gateway-firewall', 'POST', { fingerprint: preview!.fingerprint, confirm: true })
      .then(answer => { setResult(answer); setConfirming(false); check(); })
      .catch(error => setProblem(error.message)).finally(() => setBusy(false));
  };
  if (problem) return <p className="small gateway-warning" role="alert" style={{ margin: 0 }}>{problem}</p>;
  if (!preview) return <p className="small muted" style={{ margin: 0 }}>{COPY.checking}</p>;
  if (preview.open) return <p className="small row" style={{ margin: 0 }}>
    <ShieldCheck size={16} aria-hidden="true" /><span>✓ {COPY.open(port)}{result ? ` ${result.message}` : ''}</span></p>;
  return <div className="gateway-firewall" role="group" aria-label={COPY.title}>
    <p className="small gateway-warning label-row" style={{ margin: 0 }}><ShieldAlert size={16} aria-hidden="true" />
      <strong>⚠ {COPY.closed}</strong><HelpTip text={`${preview.blocked ? `${COPY.blocked} ` : ''}${COPY.why}`} label="Why?" /></p>
    <details className="side-details small"><summary>{COPY.changes}</summary><ul>
      {preview.items.filter(item => item.action === 'add').map(item => <li key={item.description}>{item.description}</li>)}
    </ul></details>
    {result && !result.applied && <ul className="small gateway-warning">
      {result.results.filter(entry => !entry.ok).map(entry => <li key={entry.description}>{entry.description}: {entry.message}</li>)}</ul>}
    {!confirming ? <button onClick={() => setConfirming(true)}>{COPY.review}</button>
      : <div className="gateway-firewall-confirm"><span className="small">{COPY.admin}</span>
        <button className="primary" disabled={busy} onClick={apply}>{COPY.apply}</button>
        <button className="ghost" disabled={busy} onClick={() => setConfirming(false)}>{COPY.cancel}</button></div>}
  </div>;
}
