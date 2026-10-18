import type { PlanPreview } from '../../api/network';
import { addressStatus } from '../../copy/status';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';

export type Manual = Record<string, { ip: string; prefix: number }>;

/** One row per IED ScadaBench publishes: the address it will answer at and whether that address is free.
 *  The SCD address is used by default; a row with a problem offers one free address to use instead. */
export function AddressTable({ preview, manual, setManual }: {
  preview: PlanPreview; manual: Manual; setManual: (value: Manual) => void;
}) {
  const type = (id: string, ip: string, prefix: number) => setManual({ ...manual, [id]: { ip, prefix } });
  return <div className="card"><table className="address-table">
    <thead><tr><th>IED</th><th>Will answer at</th><th>Check</th></tr></thead>
    <tbody>{preview.rows.map(row => {
      const typed = manual[row.ied_id];
      const prefix = typed?.prefix ?? row.prefix ?? 24;
      const status = addressStatus(row.status);
      return <tr key={row.ied_id}>
        <td><strong>{row.name}</strong></td>
        <td><div className="row" style={{ gap: 8 }}>
          <input aria-label={`Address for ${row.name}`} style={{ width: 160 }} value={typed?.ip ?? row.proposed}
            onChange={event => type(row.ied_id, event.target.value, prefix)} />
          <span className="small muted">{typed ? 'typed by you'
            : row.scd_ip && row.proposed === row.scd_ip ? <>from the <InfoTip term="scd" label="SCD" /></> : 'free address'}</span>
        </div></td>
        <td><div className="address-check">
          <StatusPill status={status} />
          {(row.hint || status.hint) && <span className="small muted">{row.hint || status.hint}</span>}
          {row.alternative && <button className="link-button" onClick={() =>
            type(row.ied_id, row.alternative!.ip, row.alternative!.prefix)}>Use {row.alternative.ip} instead</button>}
        </div></td>
      </tr>;
    })}</tbody></table></div>;
}
