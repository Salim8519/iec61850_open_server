import { useEffect, useState } from 'react';
import type { GatewayPoint } from '../../../api/gateway';
import { OWN_COPY } from '../../../copy/gateway';
import { SegmentedControl } from '../../../ui/SegmentedControl';
import { isCommand, isMeasured, ownName, ownTargets, parseOwnNumber, statusChoices, statusNumber, targetName } from './ownPoints';

/** The "IED · reference" cell of an own point: no IED, and what a command drives. */
export function OwnReference({ row, rows, locked, change }: {
  row: GatewayPoint; rows: GatewayPoint[]; locked: boolean; change: (target: string | undefined) => void;
}) {
  const drives = targetName(rows, row.target);
  return <td className="reference-cell" title={OWN_COPY.hint}><strong>{OWN_COPY.ownPoint}</strong>
    {isCommand(row.kind) && !locked
      ? <div><select className="own-target" value={row.target ?? ''} aria-label={OWN_COPY.drives}
        onChange={event => change(event.target.value || undefined)}>
        <option value="">{OWN_COPY.drivesNone}</option>
        {ownTargets(rows).map(choice => <option key={choice.value} value={choice.value}>{choice.label}</option>)}
      </select></div>
      : <div className="small muted own-note">{drives ? OWN_COPY.driving(drives) : OWN_COPY.madeUp}</div>}</td>;
}

/** The value an own status or measured point serves, changeable at any time. */
export function OwnValue({ row, set, fail }: { row: GatewayPoint; set: (value: number) => void; fail: (message: string) => void }) {
  const label = `Value of ${ownName(row)}`;
  if (isMeasured(row.kind)) return <OwnNumber row={row} label={label} set={set} fail={fail} />;
  const current = String(statusNumber(row.kind, row.value));
  if (row.kind === 'single') return <SegmentedControl label={label} value={current} options={statusChoices(row.kind)}
    onChange={value => set(Number(value))} />;
  return <select className="own-status" value={current} aria-label={label} onChange={event => set(Number(event.target.value))}>
    {statusChoices(row.kind).map(choice => <option key={choice.value} value={choice.value}>{choice.label}</option>)}</select>;
}

/** A number box that sends its value on Enter or when it loses focus. */
function OwnNumber({ row, label, set, fail }: { row: GatewayPoint; label: string; set: (value: number) => void;
  fail: (message: string) => void }) {
  const shown = row.value === undefined ? '0' : String(Number(row.value));
  const [text, setText] = useState(shown);
  useEffect(() => setText(shown), [shown]);
  const commit = () => {
    if (text.trim() === shown) return;
    const parsed = parseOwnNumber(text, row.kind);
    if ('error' in parsed) { fail(parsed.error); setText(shown); return; }
    set(parsed.value);
  };
  const scaled = row.scale !== 1 && row.value_text ? ` · SCADA gets ${row.value_text}` : '';
  return <input className="gateway-number" inputMode="decimal" value={text} aria-label={label}
    title={`${OWN_COPY.scaleHint}${scaled}`} onChange={event => setText(event.target.value)} onBlur={commit}
    onKeyDown={event => { if (event.key === 'Enter') commit(); if (event.key === 'Escape') setText(shown); }} />;
}
