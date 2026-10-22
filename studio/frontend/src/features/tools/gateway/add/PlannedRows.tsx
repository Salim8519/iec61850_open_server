import type { Protocol } from '../../../../api/gateway';
import type { Plan, PlannedRow } from '../../../../api/gatewayAdd';
import { ADDRESS_LABEL } from '../../../../copy/gateway';
import { ADD_COPY } from '../../../../copy/gatewayAdd';
import { kindOfGatewayPoint } from '../../../../points/pointKind';
import { KindBadge, kindRow } from '../../../../ui/KindBadge';

export type RowEditing = { name: (row: PlannedRow, text: string) => void; address: (row: PlannedRow, text: string) => void };

/** What will be added, one row per point, before anything changes. A row that cannot be added is marked
 *  with a ⚠ and its reason in words (never colour alone); "Add" stays off until every row is right.
 *  With `editing`, names and addresses can be changed here (a status point follows its command). */
export function PlannedRows({ plan, protocol, editing, numbered }: {
  plan: Plan; protocol: Protocol; editing?: RowEditing; numbered?: boolean;
}) {
  return <div className="table-scroll planned-rows"><table className="compact-rows"><thead><tr>
    {numbered && <th>{ADD_COPY.row}</th>}<th>{ADD_COPY.name}</th><th>{ADD_COPY.type}</th>
    <th>{ADDRESS_LABEL[protocol]}</th><th>{ADD_COPY.ied} · {ADD_COPY.point}</th></tr></thead>
    <tbody>{plan.rows.map((row, at) => {
      const kind = kindOfGatewayPoint(row.point.kind);
      const columns = numbered ? 5 : 4;
      return [<tr key={`${at}-point`} className={`kind-edge${row.problem ? ' planned-bad' : ''}`} {...kindRow(kind)}>
        {numbered && <td>{row.number}</td>}
        <td className="planned-name">{editing && !row.status_for
          ? <input aria-label={`${ADD_COPY.name} ${row.point.ref || at + 1}`} value={row.point.description} maxLength={120}
            onChange={event => editing.name(row, event.target.value)} />
          : <span>{row.point.description || row.cells?.name || '—'}</span>}</td>
        <td><KindBadge kind={kind} showName /></td>
        <td>{editing && !row.status_for
          ? <input className="gateway-number" inputMode="numeric" aria-label={`${ADD_COPY.address} ${row.point.ref || at + 1}`}
            defaultValue={row.point.address ?? ''} key={row.point.address} onBlur={event => editing.address(row, event.target.value)}
            onKeyDown={event => { if (event.key === 'Enter') editing.address(row, (event.target as HTMLInputElement).value); }} />
          : <span>{row.point.address ?? row.cells?.address ?? ''}</span>}</td>
        <td className="planned-ref" title={row.point.ref}>
          <strong>{row.point.ied_name || row.cells?.ied || ''}</strong>{' '}
          <span className="reference-path">{row.status_for ? `↳ ${ADD_COPY.statusOf} · ` : ''}{row.point.ref || row.cells?.point || ''}</span>
        </td></tr>,
        row.problem && <tr key={`${at}-problem`} className="planned-problem"><td colSpan={columns} role="alert">⚠ {row.problem}</td></tr>];
    })}</tbody></table></div>;
}
