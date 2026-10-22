import { Pencil, X } from 'lucide-react';
import type { GatewayPoint, PointKind, Protocol } from '../../../api/gateway';
import { CHANGE_COPY, GATEWAY_COPY, KIND_LABELS, KIND_WHERE, OWN_COPY } from '../../../copy/gateway';
import { kindOfGatewayPoint } from '../../../points/pointKind';
import { EditableText } from '../../../ui/EditableText';
import { KindBadge, kindRow } from '../../../ui/KindBadge';
import { ChoiceEdit, type EditChoice } from './ChoiceEdit';
import { CommandButtons, type PendingCommand } from './CommandSend';
import { blockedReason } from './gatewayCommands';
import { OwnReference, OwnValue } from './OwnCells';
import { isOwn, ownName } from './ownPoints';
import { parseAddress, parseScale } from './pointEdits';

const shortPath = (ref: string, ied?: string | null) => (ied && ref.startsWith(ied) && ref.length > ied.length ? ref.slice(ied.length) : ref);

/** One point on one line (ADR-0095): the type badge gives the coloured left edge; name, address and
 *  scale are text with a ✎ pin that turns into an input on click. Every change goes through `update`,
 *  exactly like the always-open inputs did; the table saves the list after each change. */
export function PointRow({ row, protocol, rows, locked, dirty, picking, picked, changing, choices, togglePicked, toggleChange,
  update, remove, setOwn, ask, fail }: {
  row: GatewayPoint; protocol: Protocol; rows: GatewayPoint[]; locked: boolean; dirty: boolean; picking: boolean;
  picked: boolean; changing: boolean; choices: EditChoice[]; togglePicked: () => void; toggleChange: () => void;
  update: (change: Partial<GatewayPoint>) => void; remove: () => void; setOwn: (value: number) => void;
  ask: (pending: PendingCommand) => void; fail: (message: string) => void;
}) {
  const own = isOwn(row);
  const command = row.kind.startsWith('command');
  const kind = kindOfGatewayPoint(row.kind);
  const title = own ? ownName(row) : row.ref;
  // The IED name already stands in bold before the path, so the path drops it when it starts with
  // it (display only; the full reference is the cell's hover text).
  return <tr {...kindRow(kind)} className={picked ? 'picked' : undefined}>
    {picking && <td><input type="checkbox" checked={picked} aria-label={`Tick ${title}`} onChange={togglePicked} /></td>}
    <td className="point-name"><EditableText value={row.description} disabled={locked} label={GATEWAY_COPY.name}
      placeholder={own ? OWN_COPY.namePlaceholder : GATEWAY_COPY.namePlaceholder} onSave={description => update({ description })} /></td>
    <td><ChoiceEdit label="Point type" value={row.kind} choices={choices} disabled={locked}
      shown={<span className="type-cell" title={`${KIND_LABELS[row.kind]} (${KIND_WHERE[protocol][row.kind]})`}>
        <KindBadge kind={kind} raw={KIND_WHERE[protocol][row.kind]} /></span>}
      onChange={next => update(own ? { kind: next as PointKind, value: undefined, target: undefined } : { kind: next as PointKind })} /></td>
    <td className="address-cell"><EditableText mono inputMode="numeric" value={String(row.address)} disabled={locked} label="Address"
      onSave={text => update({ address: parseAddress(text) })} /></td>
    {own ? <OwnReference row={row} rows={rows} locked={locked} change={target => update({ target })} />
      : <td className="reference-cell" title={row.ref}><span className="reference-line"><strong>{row.ied_name ?? row.ied}</strong>
        <span className="small mono reference-path">{shortPath(row.ref, row.ied_name)}</span>
        {!locked && <button type="button" className="ghost reference-change" aria-expanded={changing}
          title={CHANGE_COPY.button} aria-label={CHANGE_COPY.button} onClick={toggleChange}><Pencil size={13} />
          <span className="reference-change-word">{CHANGE_COPY.short}</span></button>}</span>
        {row.monitor_only && command && <div className="small gateway-warning">Real IED: commands are refused</div>}</td>}
    <td className="scale-cell"><EditableText mono inputMode="decimal" value={String(row.scale)} disabled={locked} label="Scale"
      onSave={text => update({ scale: parseScale(text) })} /></td>
    <td className={`small value-cell${command ? ' command-cell' : ''}`} title={command && !own ? undefined : row.value_text}>{command
      ? <div className="own-command"><CommandButtons row={row} blocked={blockedReason(row, dirty)} ask={ask} />
        {own && <span className="small muted">{row.value_text ?? 'No command yet'}</span>}</div>
      : own ? <OwnValue row={row} set={setOwn} fail={fail} />
        : <span className={row.online ? '' : 'muted'}>{row.online === false && '⚠ '}{row.value_text ?? '—'}</span>}</td>
    <td><button className="ghost icon-button" disabled={locked} aria-label={`Remove ${own && !row.description.trim() ? `the point at address ${row.address}` : title}`} title="Remove"
      onClick={remove}><X size={15} /></button></td>
  </tr>;
}
