import { useEffect, useState } from 'react';
import { ShoppingCart } from 'lucide-react';
import { buyUrl, getLicenseStatus, type LicenseStatus } from '../../api/license';
import { licenseState } from '../../copy/license';
import { StatusPill } from '../../ui/StatusPill';
import { Spinner } from '../../ui/Spinner';
import { KeyActivation, ServerOptions } from '../license/LicenseActions';
import { VM_HOST_COPY } from '../../copy/vmHost';
import { VmHostSection } from './VmHostSection';

/** Settings › License: current state, Machine ID, server address and upgrade from trial. */
export function LicenseSection() {
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const [error, setError] = useState('');
  useEffect(() => { getLicenseStatus().then(setStatus).catch(reason => setError(String(reason.message || reason))); }, []);
  if (error) return <div className="card"><div className="card-body"><p className="muted">{error}</p></div></div>;
  if (!status) return <div className="card"><div className="card-body"><Spinner label="Reading the license…" /></div></div>;
  const state = licenseState(status);
  const purchase = buyUrl(status);
  const rows: [string, string][] = [['Status', state.label]];
  if (status.app_version) rows.push(['Version', status.app_version]);
  if (status.masked_key) rows.push(['License key', status.masked_key]);
  if (status.email) rows.push(['Licensed to', status.email]);
  if (status.host_key) rows.push([VM_HOST_COPY.hostKeyRow, status.host_key]);
  if (status.expires_at) rows.push(['Trial ends', new Date(status.expires_at * 1000).toLocaleString()]);
  return <div className="card">
    <div className="card-heading"><h2>License</h2><StatusPill status={state} /></div>
    <table><tbody>{rows.map(([label, value]) => <tr key={label}><td className="muted" style={{ width: 240 }}>{label}</td>
      <td><strong>{value}</strong></td></tr>)}</tbody></table>
    <div className="card-body">
      <p className="small muted" style={{ margin: 0 }}>{state.hint}</p>
      {status.mode === 'trial' && <>
        {purchase && <a className="button" href={purchase} target="_blank" rel="noreferrer" style={{ justifySelf: 'start' }}>
          <ShoppingCart size={16} aria-hidden />Buy lifetime license</a>}
        <KeyActivation changed={setStatus} compact />
      </>}
      {status.enforced && status.mode === 'lifetime' && <VmHostSection />}
      {status.enforced && <ServerOptions status={status} changed={setStatus} />}
    </div>
  </div>;
}
