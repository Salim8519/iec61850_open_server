import { Check, Copy } from 'lucide-react';
import { useState } from 'react';
import { api } from '../../api/client';
import type { LiveSession } from '../../api/network';
import type { Ied, Project } from '../../api/types';
import { Disclosure } from '../../ui/Disclosure';
import { InfoTip } from '../../ui/InfoTip';
import { endpointOf } from '../network/endpoints';
import { DeviceConnection } from './DeviceConnection';
import { TestModeCard } from './TestModeCard';
import type { ModeSummary } from './useModes';

export function DeviceOverview({ ied, project, locked, session, perform, mode }: {
  ied: Ied; project: Project; locked: boolean; session?: LiveSession; mode?: ModeSummary;
  perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const [copied, setCopied] = useState(false);
  const endpoint = endpointOf(ied, project.network);
  const lab = project.network.mode === 'lab';
  const publishes = ied.goose?.publishes?.length || 0;
  const facts: [string, string][] = [
    ['Data points', ied.signal_count.toLocaleString()], ['Controls', String(ied.controls.length)],
    ['Reports', String(ied.reports.length)], ['GOOSE blocks', String(publishes)],
    ['Address in the SCD', ied.scl_address?.ip || 'None'],
  ];
  return <div className="card-body">
    {ied.role === 'simulated' && <div className="mode-banner">
      <div><h3 style={{ margin: 0 }}>Clients connect to</h3>
        <code style={{ fontSize: 18 }}>{endpoint.host}:{endpoint.port}</code>
        <p className="small muted" style={{ margin: '4px 0 0' }}>{lab ? 'On the lab network adapter.'
          : 'Only programs on this PC can reach it. Use Settings › Network to put it on a real network.'}</p></div>
      <button onClick={async () => { await navigator.clipboard.writeText(`${endpoint.host}:${endpoint.port}`); setCopied(true); }}>
        {copied ? <Check size={16} /> : <Copy size={16} />}{copied ? 'Copied' : 'Copy address'}</button>
    </div>}
    {ied.role === 'real' && <DeviceConnection ied={ied} session={session} locked={locked} perform={perform} />}
    {ied.role === 'off' && <p className="muted" style={{ margin: 0 }}>This device is switched off. It is ignored when you press Run.</p>}
    <TestModeCard ied={ied} summary={mode} perform={perform} />
    <div className="facts">{facts.map(([label, value]) => <div className="fact" key={label}><span>{label}</span><strong>{value}</strong></div>)}</div>
    {ied.role === 'simulated' && publishes > 0 && <label className="row" style={{ fontWeight: 500, color: 'var(--text)' }}>
      <input type="checkbox" checked={ied.goose_publish} disabled={locked}
        onChange={event => perform(() => api(`/ieds/${ied.id}/settings`, 'PUT', { goose_publish: event.target.checked }))} />
      <span>Send its <InfoTip term="goose" label="GOOSE" /> messages{lab ? '' : ' (works on the lab network only)'}</span></label>}
    {(!!ied.warnings.length || !!ied.goose?.diagnostics?.length || ied.validation) && <Disclosure title={
      `File check details${ied.warnings.length + (ied.goose?.diagnostics?.length || 0) ? ` · ${ied.warnings.length + (ied.goose?.diagnostics?.length || 0)} notes` : ''}`}>
      {ied.validation && <p className="small" style={{ margin: 0 }}>File structure: <strong>{ied.validation.status}</strong>
        {ied.validation.schema ? ` (${ied.validation.schema})` : ''}</p>}
      {[...ied.warnings, ...(ied.goose?.diagnostics || [])].map(note => <p key={note} className="small" style={{ margin: 0 }}>• {note}</p>)}
      {ied.validation?.errors.map((issue, index) => <p className="small" key={index} style={{ margin: 0 }}>Line {issue.line}: {issue.message}</p>)}
    </Disclosure>}
  </div>;
}
