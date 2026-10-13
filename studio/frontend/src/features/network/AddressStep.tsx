import { FileText, Shuffle } from 'lucide-react';
import type { PlanPreview } from '../../api/network';
import { Hint } from '../../ui/Hint';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import { addressStatus } from '../../copy/status';

export type Manual = Record<string, { ip: string; prefix: number }>;

export function AddressStep({ strategy, setStrategy, preview, manual, setManual, checking }: {
  strategy: 'scd' | 'adapter'; setStrategy: (value: 'scd' | 'adapter') => void; preview: PlanPreview | null;
  manual: Manual; setManual: (value: Manual) => void; checking: boolean;
}) {
  return <div className="stack">
    <h2 style={{ margin: 0 }}>Which address should each IED use?</h2>
    <div className="choice-grid">
      <button type="button" className={`choice${strategy === 'scd' ? ' selected' : ''}`} onClick={() => setStrategy('scd')}
        aria-pressed={strategy === 'scd'}>
        <span className="row"><FileText size={16} /><strong>Use the addresses from the SCD (recommended)</strong></span>
        <Hint id="network.strategy.scd" compact />
      </button>
      <button type="button" className={`choice${strategy === 'adapter' ? ' selected' : ''}`} onClick={() => setStrategy('adapter')}
        aria-pressed={strategy === 'adapter'}>
        <span className="row"><Shuffle size={16} /><strong>Pick free addresses on this adapter</strong></span>
        <Hint id="network.strategy.adapter" compact />
      </button>
    </div>
    <Hint id="network.address.manual" />
    {checking && <p className="small muted">Checking the addresses, including whether another device already uses them…</p>}
    {preview && <div className="card"><table>
      <thead><tr><th>IED</th><th>Address in <InfoTip term="scd" label="SCD" /></th><th>Address to use</th><th>Check</th></tr></thead>
      <tbody>{preview.rows.map(row => {
        const typed = manual[row.ied_id];
        return <tr key={row.ied_id}>
          <td><strong>{row.name}</strong></td>
          <td>{row.scd_ip ? <code>{row.scd_ip}</code> : <span className="muted">None</span>}</td>
          <td><div className="row">
            <input aria-label={`Address for ${row.name}`} style={{ width: 150 }}
              value={typed?.ip ?? row.proposed} onChange={event => setManual({ ...manual,
                [row.ied_id]: { ip: event.target.value, prefix: typed?.prefix ?? row.prefix ?? 24 } })} />
            <span className="muted">/</span>
            <input aria-label={`Prefix for ${row.name}`} style={{ width: 64 }} type="number" min={8} max={30}
              value={typed?.prefix ?? row.prefix ?? 24} onChange={event => setManual({ ...manual,
                [row.ied_id]: { ip: typed?.ip ?? row.proposed, prefix: Number(event.target.value) } })} />
          </div></td>
          <td><StatusPill status={{ ...addressStatus(row.status), hint: row.hint || addressStatus(row.status).hint }} showHint /></td>
        </tr>;
      })}</tbody></table></div>}
  </div>;
}
