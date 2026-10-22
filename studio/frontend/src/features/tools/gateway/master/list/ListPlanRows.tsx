import type { MasterProtocol } from '../../../../../api/master';
import type { ListPlan, MasterType } from '../../../../../api/masterList';
import { ADD_LIST_COPY, LIST_ADDRESS } from '../../../../../copy/masterList';
import { KindBadge, kindRow } from '../../../../../ui/KindBadge';
import { kindOfListType } from './listRows';

/** What will be added, one row per point, before anything changes. A row that cannot be added is marked
 *  with ⚠ and its reason in words (never colour alone); Add stays off until every row is right. */
export function ListPlanRows({ plan, protocol, types, numbered }: {
  plan: ListPlan; protocol: MasterProtocol; types: MasterType[]; numbered?: boolean;
}) {
  const byKind = new Map(types.map(type => [type.kind, type]));
  const columns = numbered ? 4 : 3;
  return <div className="table-scroll planned-rows master-plan"><table className="compact-rows"><thead><tr>
    {numbered && <th>{ADD_LIST_COPY.row}</th>}<th>{ADD_LIST_COPY.name}</th><th>{ADD_LIST_COPY.type}</th>
    <th>{LIST_ADDRESS[protocol]}</th></tr></thead>
    <tbody>{plan.rows.map((row, at) => {
      const type = byKind.get(row.point.type);
      const kind = kindOfListType(type);
      return [<tr key={`${at}-point`} className={`kind-edge${row.problem ? ' planned-bad' : ''}`} {...kindRow(kind)}>
        {numbered && <td>{row.number}</td>}
        <td className="planned-name"><span>{row.point.name || row.cells?.name || '—'}</span></td>
        <td><span className="type-cell"><KindBadge kind={kind} raw={type?.type_id} />
          <span className="small">{type?.name ?? row.cells?.type ?? row.point.type}</span></span></td>
        <td className="mono">{row.point.address ?? row.cells?.address ?? ''}</td></tr>,
        row.problem && <tr key={`${at}-problem`} className="planned-problem"><td colSpan={columns} role="alert">⚠ {row.problem}</td></tr>];
    })}</tbody></table></div>;
}
