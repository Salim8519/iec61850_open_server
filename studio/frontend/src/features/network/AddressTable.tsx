import type { PlanPreview } from '../../api/network';
import { addressStatus } from '../../copy/status';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import type { Provisional } from './typedAddressCheck';

/** The address the IED's file (CID/SCD) gives, always in view (change 0150), with one click to
 *  go back to it after typing another address or taking a free one. */
function FileAddress({ row, shown, use }: { row: PlanPreview['rows'][number]; shown: string; use: () => void }) {
  if (!row.scd_ip) return row.status === 'missing_scd_address' ? null
    : <div className="small muted file-address">No address in the file</div>;
  return <div className="small muted file-address">File address: <strong>{row.scd_ip}</strong>
    {shown !== row.scd_ip && <button className="link-button" onClick={use}>Use file address</button>}</div>;
}

export type Manual = Record<string, { ip: string; prefix: number }>;

/** One row per IED (and SNMP device, ADR-0092) ScadaBench publishes: the address it will answer at and whether
 *  that address is free. The SCD address is used by default; SNMP devices keep their saved address while it is
 *  free; a row with a problem offers one free address to use instead. A row just typed in shows its first answer
 *  (`provisional`) at once, until the engine's own answer arrives. */
export function AddressTable({ preview, manual, setManual, provisional = {} }: {
  preview: PlanPreview; manual: Manual; setManual: (value: Manual) => void; provisional?: Record<string, Provisional>;
}) {
  const type = (id: string, ip: string, prefix: number) => setManual({ ...manual, [id]: { ip, prefix } });
  return <div className="card"><table className="address-table">
    <thead><tr><th>{preview.rows.some(row => row.kind === 'snmp') ? 'Device' : 'IED'}</th><th>Will answer at</th><th>Check</th></tr></thead>
    <tbody>{preview.rows.map(row => {
      const typed = manual[row.ied_id];
      const prefix = typed?.prefix ?? row.prefix ?? 24;
      const first = provisional[row.ied_id];
      const status = addressStatus(first?.status ?? row.status);
      return <tr key={row.ied_id}>
        <td><strong>{row.name}</strong></td>
        <td><div className="row" style={{ gap: 8 }}>
          <input aria-label={`Address for ${row.name}`} style={{ width: 160 }} value={typed?.ip ?? row.proposed}
            onChange={event => type(row.ied_id, event.target.value, prefix)} />
          <span className="small muted">{typed ? 'typed by you' : row.saved ? 'saved address'
            : row.scd_ip && row.proposed === row.scd_ip ? <>from the <InfoTip term="scd" label="SCD" /></> : 'free address'}</span>
        </div>
          {row.kind !== 'snmp' && <FileAddress row={row} shown={typed?.ip ?? row.proposed}
            use={() => type(row.ied_id, row.scd_ip, prefix)} />}</td>
        <td><div className="address-check">
          <StatusPill status={status} />
          {(first?.hint || row.hint || status.hint) && <span className="small muted">{first?.hint || row.hint || status.hint}</span>}
          {!first && row.alternative && <button className="link-button" onClick={() =>
            type(row.ied_id, row.alternative!.ip, row.alternative!.prefix)}>Use {row.alternative.ip} instead</button>}
        </div></td>
      </tr>;
    })}</tbody></table></div>;
}
