import { useEffect, useMemo, useState } from 'react';
import { gatewayPointsPath, type GatewayPoint, type Protocol } from '../../../../api/gateway';
import { liveFeed } from '../../../../api/polls';
import { ADDRESS_LABEL, GATEWAY_COPY, KIND_LABELS, KIND_WHERE } from '../../../../copy/gateway';
import { CONVERTER_COPY } from '../../../../copy/converter';
import { kindOfGatewayPoint } from '../../../../points/pointKind';
import { useListFilter } from '../../../../state/useListFilter';
import { FilterBar } from '../../../../ui/FilterBar';
import { HelpTip } from '../../../../ui/HelpTip';
import { KindBadge, kindRow } from '../../../../ui/KindBadge';
import { kindCounts } from '../kindCounts';
import { SERVED_POINT_FILTER, servedPointMatches } from '../pointFilters';

const SHOWN = 500;

/** What SCADA sees: the device points with their upstream addresses (read only; tick in Device points). */
export function ServedPoints({ protocol, fail }: { protocol: Protocol; fail: (message: string) => void }) {
  const [rows, setRows] = useState<GatewayPoint[]>([]);
  const filter = useListFilter('served-points', SERVED_POINT_FILTER);
  useEffect(() => liveFeed.subscribe<{ points: GatewayPoint[] }>(gatewayPointsPath(protocol), 1000,
    value => setRows(value.points), error => fail(error.message)), [protocol]);
  const shown = useMemo(() => rows.filter(row => servedPointMatches(row, filter.values)), [rows, filter.values]);
  const counts = useMemo(() => kindCounts(rows, row => kindOfGatewayPoint(row.kind)), [rows]);
  return <>
    <FilterBar label="Filter served points" spec={SERVED_POINT_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={rows.length} counts={counts} extra={<HelpTip text={CONVERTER_COPY.servedHint} label="What is this?" />}
      note={shown.length > SHOWN ? `(first ${SHOWN} shown)` : undefined} />
    {rows.length === 0 ? <p className="muted watch-empty">{CONVERTER_COPY.servedEmpty}</p> :
      <div className="table-scroll"><table className="gateway-points compact-rows"><thead><tr><th>Upstream type</th>
        <th>{ADDRESS_LABEL[protocol]}</th><th>Device point</th><th>Value now</th></tr></thead>
        <tbody>{shown.slice(0, SHOWN).map(row => {
          const kind = kindOfGatewayPoint(row.kind);
          return <tr key={row.id} {...kindRow(kind)}>
            <td><span className="type-cell"><KindBadge kind={kind} raw={KIND_WHERE[protocol][row.kind]} />
              <span className="small">{KIND_LABELS[row.kind]}</span></span></td>
            <td className="mono">{row.address}</td>
            <td>{row.description}</td>
            <td className={row.online ? '' : 'muted'}>{row.value_text ?? '—'}</td>
          </tr>;
        })}</tbody></table>
        {shown.length === 0 && <p className="muted watch-empty">{GATEWAY_COPY.emptyFilter}</p>}</div>}
  </>;
}
