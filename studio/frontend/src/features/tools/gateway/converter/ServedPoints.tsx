import { useEffect, useState } from 'react';
import { gatewayPointsPath, type GatewayPoint, type Protocol } from '../../../../api/gateway';
import { liveFeed } from '../../../../api/polls';
import { ADDRESS_LABEL, KIND_LABELS, KIND_WHERE } from '../../../../copy/gateway';
import { CONVERTER_COPY } from '../../../../copy/converter';
import { HelpTip } from '../../../../ui/HelpTip';

/** What SCADA sees: the device points with their upstream addresses (read only; tick in Device points). */
export function ServedPoints({ protocol, fail }: { protocol: Protocol; fail: (message: string) => void }) {
  const [rows, setRows] = useState<GatewayPoint[]>([]);
  useEffect(() => liveFeed.subscribe<{ points: GatewayPoint[] }>(gatewayPointsPath(protocol), 1000,
    value => setRows(value.points), error => fail(error.message)), [protocol]);
  return <>
    <div className="panel-toolbar"><span className="small">{rows.length > 500 ? `First 500 of ${rows.length} points`
      : `${rows.length} points`}</span><HelpTip text={CONVERTER_COPY.servedHint} label="What is this?" /></div>
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
