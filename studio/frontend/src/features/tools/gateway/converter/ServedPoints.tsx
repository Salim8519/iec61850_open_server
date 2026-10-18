import { useEffect, useState } from 'react';
import { gatewayApi, type GatewayPoint, type Protocol } from '../../../../api/gateway';
import { ADDRESS_LABEL, KIND_LABELS, KIND_WHERE } from '../../../../copy/gateway';
import { CONVERTER_COPY } from '../../../../copy/converter';

/** What SCADA sees: the device points with their upstream addresses (read only; tick in Device points). */
export function ServedPoints({ protocol, fail }: { protocol: Protocol; fail: (message: string) => void }) {
  const [rows, setRows] = useState<GatewayPoint[]>([]);
  useEffect(() => {
    const load = () => gatewayApi.points(protocol).then(value => setRows(value.points)).catch(error => fail(error.message));
    load();
    const timer = setInterval(load, 1000);
    return () => clearInterval(timer);
  }, [protocol]);
  return <>
    <div className="guide-section"><p className="small muted" style={{ margin: 0 }}>{CONVERTER_COPY.servedHint}</p></div>
    {rows.length === 0 ? <p className="muted watch-empty">{CONVERTER_COPY.servedEmpty}</p> :
      <div className="table-scroll"><table className="gateway-points"><thead><tr><th>Upstream type</th>
        <th>{ADDRESS_LABEL[protocol]}</th><th>Device point</th><th>Value now</th></tr></thead>
        <tbody>{rows.slice(0, 500).map(row => <tr key={row.id}>
          <td><span className="type-cell">{KIND_LABELS[row.kind]}<span className="small muted">{KIND_WHERE[protocol][row.kind]}</span></span></td>
          <td className="mono">{row.address}</td>
          <td>{row.description}</td>
          <td className={row.online ? '' : 'muted'}>{row.value_text ?? '—'}</td>
        </tr>)}</tbody></table></div>}
  </>;
}
