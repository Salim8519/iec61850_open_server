import { Check, Copy, Network } from 'lucide-react';
import { useState } from 'react';
import type { Ied } from '../../api/types';

export function Connections({ ieds }: { ieds: Ied[] }) {
  const [copied, setCopied] = useState('');
  return <div className="main-view"><div className="view-heading"><div><h1>Connections</h1>
    <p className="muted small">Connect IEDScout or your SCADA client to these local MMS endpoints.</p></div>
    <span className="badge"><Network size={13} />Local loopback</span></div>
    <div className="content-page" style={{ paddingTop: 0 }}><div className="card">
      <table><thead><tr><th>IED</th><th>IP address</th><th>MMS port</th><th>State</th><th /></tr></thead>
        <tbody>{ieds.map(ied => <tr key={ied.id}><td>{ied.name}</td><td><code>{ied.ip}</code></td><td><code>{ied.port}</code></td>
          <td><span className={`badge ${ied.state === 'running' ? 'live' : ''}`}><span className="dot" />{ied.state}</span></td>
          <td><button onClick={async () => { await navigator.clipboard.writeText(`${ied.ip}:${ied.port}`); setCopied(ied.id); }}>
            {copied === ied.id ? <Check size={14} /> : <Copy size={14} />}Copy</button></td></tr>)}</tbody></table>
      {!ieds.length && <div className="empty">Add an IED to allocate its loopback endpoint.</div>}
    </div><div className="intro-card" style={{ maxWidth: 'none', marginTop: 20, padding: 20 }}>
      <h2>Lab network</h2><p style={{ marginBottom: 0 }}>Guided LAN adapter setup is still being implemented. This build opens only loopback endpoints and makes no adapter or firewall changes.</p>
    </div></div>
  </div>;
}
