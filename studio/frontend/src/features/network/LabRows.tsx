import { Activity } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import { LAB, type LabPing, type LabRow } from '../../api/lab';
import { LAB_COPY, LAB_ROW_STATUS, pingWords } from '../../copy/labNetwork';
import { SpinIcon } from '../../ui/Spinner';
import { StatusPill } from '../../ui/StatusPill';

/** One IED: the address it answers at on this PC (typed here), the engine's verdict and a Ping. The file address
 *  shows only when another one is used; a hint only when something needs attention. `commit` sends a typed
 *  address: a check before the lab network is in use, a move once it is. */
export function LabRowView({ row, typed, setTyped, commit, busy }: {
  row: LabRow; typed: string | undefined; setTyped: (value: string | undefined) => void;
  commit: (address: string) => void; busy: boolean;
}) {
  const [ping, setPing] = useState<{ address: string; answer: LabPing | 'failed' } | null>(null);
  const [pinging, setPinging] = useState(false);
  const shown = typed ?? row.ip;
  useEffect(() => setPing(null), [row.ip]);
  const status = { ...LAB_ROW_STATUS[row.status] ?? LAB_ROW_STATUS.invalid, hint: row.hint };
  const send = () => { if (typed !== undefined && typed.trim() !== row.ip) commit(typed.trim()); };
  const doPing = () => {
    setPinging(true);
    api<LabPing>(`${LAB}/ping`, 'POST', { ip: row.ip })
      .then(answer => setPing({ address: row.ip, answer }))
      .catch(() => setPing({ address: row.ip, answer: 'failed' }))
      .finally(() => setPinging(false));
  };
  return <tr>
    <td><strong>{row.name}</strong></td>
    <td><div className="lab-address">
      <input aria-label={`Address of ${row.name} on this PC`} className="mono" value={shown} disabled={busy}
        onChange={event => setTyped(event.target.value)} onBlur={send}
        onKeyDown={event => { if (event.key === 'Enter') send(); if (event.key === 'Escape') setTyped(undefined); }} />
      {row.yours && row.scl_ip && <span className="small muted">File: {row.scl_ip} · <button className="link-button"
        disabled={busy} onClick={() => commit(row.scl_ip)}>{LAB_COPY.backToFile}</button></span>}
    </div></td>
    <td><div className="address-check">
      <StatusPill status={status} />
      {row.status !== 'ok' && row.hint && <span className="small muted">{row.hint}</span>}
      {row.alternative && <button className="link-button" disabled={busy} onClick={() => commit(row.alternative!)}>
        Use {row.alternative} instead</button>}
      {ping && <span className="small mono" role="status">{pingWords(ping.address, ping.answer)}</span>}
    </div></td>
    <td><button disabled={!row.on_card || pinging} onClick={doPing}
      title={row.on_card ? `Ping ${row.ip} from this PC` : 'Ping works once the address is on the lab network card'}>
      {pinging ? <SpinIcon size={14} /> : <Activity size={14} />}{LAB_COPY.ping}</button></td>
  </tr>;
}
