import type { Ied } from '../../api/types';
import { externalClients } from '../../copy/signalLayers';
import { useMmsClients } from './useGooseLinks';

/** Overview table of the SCADA or test programs that read this simulated IED over MMS (LAYER-16). */
export function ReadByClients({ ied, running }: { ied: Ied; running: boolean }) {
  const clients = externalClients(useMmsClients(ied.id, running && ied.role === 'simulated'));
  if (ied.role !== 'simulated' || !running) return null;
  return <section className="read-by-clients">
    <h3 style={{ margin: '4px 0 8px' }}>Read by clients</h3>
    {clients.length ? <div className="table-scroll"><table><thead><tr><th>Client address</th><th>Reads</th><th>Reports on</th><th>Commands</th></tr></thead>
      <tbody>{clients.map(client => <tr key={client.peer}>
        <td><code>{client.ip}</code>{client.this_pc ? ' (this PC)' : ''}{client.reading ? '' : <span className="muted"> · idle</span>}</td>
        <td>{client.reads.slice(0, 4).map(item => `${item.node.split('/').pop()} ×${item.count}`).join(', ') || '—'}
          {client.reads.length > 4 || client.more_nodes ? ' …' : ''}</td>
        <td>{client.rcbs.join(', ') || '—'}</td><td>{client.writes || '—'}</td></tr>)}</tbody></table></div>
      : <p className="small muted" style={{ margin: 0 }}>No SCADA or test program is connected to this IED over MMS right now.</p>}
  </section>;
}
