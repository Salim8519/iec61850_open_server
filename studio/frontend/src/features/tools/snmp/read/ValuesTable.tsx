import { Eye, EyeOff, PenLine, PlugZap, Star } from 'lucide-react';
import { useMemo, useState } from 'react';
import { SNMP_VALUES, snmpApi, type SnmpConfig, type SnmpStatus, type ValueRow } from '../../../../api/snmp';
import { useLiveFeed } from '../../../../api/useLiveFeed';
import { TYPE_NAMES } from '../../../../copy/snmp';
import { READ_COPY } from '../../../../copy/snmpRead';
import { useListFilter } from '../../../../state/useListFilter';
import { FilterBar } from '../../../../ui/FilterBar';
import { KindBadge, kindRow } from '../../../../ui/KindBadge';
import { Pager } from '../../gateway/Pager';
import { ValueText } from '../simulate/ValueEditor';
import { kindCounts, kindOfValue, readShown, READ_FILTER } from '../valueFilter';
import { WriteValue } from './WriteValue';

const PAGE = 100;

/** Everything the walk found, important values first (marked with a star); watched values are re-read at the
 *  chosen interval. One line per value, with the type as a badge and a chip per type to filter by. */
export function ValuesTable({ status, config, refresh, fail }: {
  status: SnmpStatus; config: SnmpConfig; refresh: () => void; fail: (message: string) => void;
}) {
  const reader = status.reader;
  const active = reader.state === 'connected' || reader.state === 'walking';
  const feed = useLiveFeed<{ values: ValueRow[] }>(active ? SNMP_VALUES : null, 1000);
  const filter = useListFilter('snmp-read-values', READ_FILTER);
  const [page, setPage] = useState(0);
  const [writing, setWriting] = useState<ValueRow | null>(null);
  const rows = useMemo(() => {
    const all = feed.value?.values ?? [];
    return [...all.filter(row => row.key), ...all.filter(row => !row.key)];
  }, [feed.value]);
  const counts = useMemo(() => kindCounts(rows), [rows]);
  const shown = useMemo(() => rows.filter(row => readShown(row, filter.values)), [rows, filter.values]);
  const first = Math.min(page, Math.max(0, Math.ceil(shown.length / PAGE) - 1)) * PAGE;
  const watched = config.reader.watched;
  const toggleWatch = (row: ValueRow) => snmpApi.watch(watched.includes(row.oid)
    ? watched.filter(oid => oid !== row.oid) : [...watched, row.oid]).then(refresh).catch(reason => fail(reason.message));
  if (!active && rows.length === 0) return <div className="panel-empty"><div className="guide-empty">
    <span className="guide-empty-icon"><PlugZap size={24} /></span>
    <h3>{READ_COPY.emptyTitle}</h3><p>{READ_COPY.emptyHint}</p></div></div>;
  return <>
    {writing && <WriteValue row={writing} close={() => setWriting(null)} fail={fail} />}
    <FilterBar label="Filter values" spec={READ_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={rows.length} counts={counts} />
    <div className="table-scroll"><table className="gateway-points snmp-values compact-rows"><thead><tr>
      <th>Name</th><th className="col-oid">OID</th><th className="col-type">Type</th><th>Value</th>
      <th className="col-actions" aria-label="Actions" /></tr></thead>
      <tbody>{shown.slice(first, first + PAGE).map(row => <tr key={row.oid} {...kindRow(kindOfValue(row))}>
        <td>{row.key && <Star className="snmp-important" size={14} role="img" aria-label={READ_COPY.important} />}{row.name}</td>
        <td className="small mono">{row.oid}</td>
        <td><KindBadge kind={kindOfValue(row)} raw={TYPE_NAMES[row.type] ?? row.type} /></td>
        <td className="snmp-value"><ValueText row={row} /></td>
        <td><span className="row snmp-row-actions">
          {!row.key && <button className="ghost" title={READ_COPY.watchHint} onClick={() => toggleWatch(row)}>
            {watched.includes(row.oid) ? <><EyeOff size={14} />{READ_COPY.unwatch}</> : <><Eye size={14} />{READ_COPY.watch}</>}
          </button>}
          {row.writable && row.type && <button className="ghost" onClick={() => setWriting(row)}>
            <PenLine size={14} />{READ_COPY.set}</button>}
        </span></td>
      </tr>)}</tbody></table>
      {rows.length > 0 && shown.length === 0 && <p className="muted watch-empty">{READ_COPY.none}</p>}</div>
    <Pager count={shown.length} page={page} size={PAGE} setPage={setPage} noun="values" />
  </>;
}
