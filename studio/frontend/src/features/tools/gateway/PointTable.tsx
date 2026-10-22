import { Undo2 } from 'lucide-react';
import { Fragment, useEffect, useMemo, useState } from 'react';
import { gatewayApi, type GatewayPoint, type PointKind, type Protocol } from '../../../api/gateway';
import type { Ied } from '../../../api/types';
import { ADDRESS_LABEL, GATEWAY_COPY, OWN_COPY } from '../../../copy/gateway';
import { ADD_COPY } from '../../../copy/gatewayAdd';
import type { SmartAnswer } from '../../../api/gatewayAdd';
import { kindOfGatewayPoint } from '../../../points/pointKind';
import { useListFilter } from '../../../state/useListFilter';
import { FilterBar } from '../../../ui/FilterBar';
import { choicesOf, effective } from '../../../ui/filterState';
import { stillThere, toggleShown, togglePick } from '../../../ui/pickRows';
import { SaveProblem } from '../../../ui/SaveStatus';
import { SelectBar } from '../../../ui/SelectBar';
import { AddPointsDialog } from './add/AddPointsDialog';
import { SmartPropose } from './add/SmartPropose';
import { CommandWarning, type PendingCommand } from './CommandSend';
import { kindCounts } from './kindCounts';
import { isOwn, liveEdit, ownName } from './ownPoints';
import { typeChoices } from './pointEdits';
import { GATEWAY_POINT_FILTER, gatewayPointMatches } from './pointFilters';
import { pointKey } from './pointRows';
import { Pager } from './Pager';
import { PointRow } from './PointRow';
import { PointToolbar } from './PointToolbar';
import { SignalChange } from './SignalChange';
import { usePointList } from './usePointList';

const PAGE = 100;

/** The SCADA point list: proposed, imported or made up (own points) and edited here. Every change is saved
 *  at once (usePointList); a whole-list action offers Undo, since there is no Save button to hold it back. */
export function PointTable({ protocol, ieds, locked, fail }: {
  protocol: Protocol; ieds: Ied[]; locked: boolean; fail: (message: string) => void;
}) {
  const { rows, setRows, dirty, problems, undo, edit, update, discard, load, state } = usePointList(protocol, locked, fail);
  const [notes, setNotes] = useState<string[]>([]);
  const [skipped, setSkipped] = useState<string[]>([]); // lines of an imported file that could not be used (not a save problem)
  const [page, setPage] = useState(0);
  const [adding, setAdding] = useState(false);
  const [proposing, setProposing] = useState(false);
  const [pending, setPending] = useState<PendingCommand | null>(null);
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState<{ ok: boolean; text: string } | null>(null);
  const [picked, setPicked] = useState<Set<string> | null>(null);
  const [changing, setChanging] = useState<string | null>(null);
  const filter = useListFilter('scada-points', GATEWAY_POINT_FILTER);
  // A command note or a pending confirmation belongs to the protocol it was made for (0178, as the master).
  useEffect(() => { setNotes([]); setSkipped([]); setAdding(false); setProposing(false); setPicked(null); setChanging(null);
    setSent(null); setPending(null); }, [protocol]);
  useEffect(() => setPage(0), [filter.values]);
  // Propose from IEDs opens the smart preview (ADR-0105): per-IED ticks, "Add only new points" keeps the list as
  // edited, "Replace" keeps own points. Applying hands over the list before, so Undo puts it back.
  const applyProposal = (value: SmartAnswer) => {
    edit(value.points, rows); setSkipped([]); setProposing(false);
    setNotes(value.notes.length ? value.notes : [GATEWAY_COPY.proposedNote]);
  };
  // Add points: the new points join the list and save at once; Undo removes them again.
  const addPoints = (points: GatewayPoint[]) => {
    if (!rows) return;
    edit([...rows, ...points], rows); setAdding(false); setSkipped([]); setNotes([ADD_COPY.added(points.length)]);
  };
  const importFile = (picked: File) => picked.text().then(text => gatewayApi.importCsv(protocol, text))
    .then(value => {
      // A file with no usable rows leaves the list as it was; only its problems are shown.
      if (value.points.length) edit(value.points, rows);
      setSkipped(value.problems); setNotes([value.points.length ? `${value.points.length} points read from ${picked.name}.`
        : `No points could be read from ${picked.name}; the list is unchanged.`]);
    })
    .catch(error => fail(error.message));
  // A saved own point changes at once (SCADA sees it while the gateway runs); while the list has changes
  // that are not saved yet, the value goes into the list and is saved with it.
  const setOwn = (index: number, row: GatewayPoint, value: number) => liveEdit(row, dirty)
    ? gatewayApi.setOwnValue(row.id!, value).then(answer => {
      setRows(current => current && current.map(item => item.id === row.id ? { ...item, ...answer.point } : item)); load();
    }).catch(error => fail(error.message))
    : update(index, { value });
  const send = (command: PendingCommand) => {
    setSending(true); setSent(null);
    const own = isOwn(command.row);
    gatewayApi.send(command.row.id!, command.value)
      .then(answer => setSent({ ok: answer.ok, text: own ? OWN_COPY.sentResult(command.label, ownName(command.row), answer.ok, answer.reason)
        : GATEWAY_COPY.sentResult(command.label, command.row.ref, answer.ok, answer.reason) }))
      .catch(error => fail(error.message)).finally(() => { setSending(false); setPending(null); load(); });
  };
  const options = useMemo(() => ({ ied: choicesOf((rows ?? []).map(row => row.ied_name ?? row.ied)) }), [rows]);
  const values = useMemo(() => effective(GATEWAY_POINT_FILTER, filter.values, options), [filter.values, options]);
  const shown = useMemo(() => (rows ?? []).map((row, index) => ({ row, index }))
    .filter(({ row }) => gatewayPointMatches(row, values)), [rows, values]);
  const first = Math.min(page, Math.max(0, Math.ceil(shown.length / PAGE) - 1)) * PAGE;
  const shownKeys = shown.map(({ row, index }) => pointKey(row, index));
  // Signals other IED points of the same type already serve: the Change row offers them as taken.
  const takenFor = (index: number, kind: PointKind) => new Set((rows ?? [])
    .filter((other, at) => at !== index && !isOwn(other) && other.kind === kind).map(other => `${other.ied}:${other.ref}`));
  const chosen = picked && rows ? stillThere(picked, rows.map(pointKey)) : [];
  const choices = useMemo(() => typeChoices(protocol), [protocol]);
  const counts = useMemo(() => kindCounts(rows ?? [], row => kindOfGatewayPoint(row.kind)), [rows]);
  // Removing is saved at once, so it always offers Undo (the list before is kept for it).
  const removeRows = (keep: (row: GatewayPoint, index: number) => boolean) => {
    if (!rows) return;
    const left = rows.filter(keep);
    edit(left, rows);
    setNotes([GATEWAY_COPY.removedNote(rows.length - left.length)]);
  };
  const removeChosen = () => { const gone = new Set(chosen); removeRows((row, index) => !gone.has(pointKey(row, index))); setPicked(null); };
  // A list that is not being saved (it has problems, or waits for the gateway to stop) can be discarded.
  const stuck = dirty && state.phase !== 'saving';
  const undoButton = undo && <button type="button" className="ghost" onClick={() => { setNotes([]); edit(undo); }}>
    <Undo2 size={14} />{GATEWAY_COPY.undo}</button>;
  return <>
    <PointToolbar protocol={protocol} locked={locked} dirty={dirty} adding={adding} hasRows={!!rows?.length} picking={!!picked}
      propose={() => setProposing(true)} add={() => setAdding(true)} togglePicking={() => setPicked(picked ? null : new Set())}
      removeAll={() => removeRows(() => false)} importFile={importFile} state={state} waiting={locked && dirty} />
    {adding && rows && <AddPointsDialog protocol={protocol} ieds={ieds} rows={rows} fail={fail} close={() => setAdding(false)}
      add={addPoints} />}
    {proposing && rows && <SmartPropose protocol={protocol} rows={rows} fail={fail} close={() => setProposing(false)}
      apply={applyProposal} />}
    <FilterBar label="Filter points" spec={GATEWAY_POINT_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={rows?.length ?? 0} options={options} counts={counts} />
    {picked && !locked && <SelectBar count={chosen.length} shown={shown.length} noun="point" cancel={() => setPicked(null)}
      selectShown={() => setPicked(toggleShown(new Set(), shownKeys))} remove={removeChosen} />}
    {(pending || sent || undo || stuck || notes.length > 0 || skipped.length > 0 || problems.length > 0) && <div className="guide-section panel-messages">
      {pending && <CommandWarning pending={pending} real={pending.row.ied_role === 'real'} own={isOwn(pending.row)} busy={sending}
        cancel={() => setPending(null)} send={() => send(pending)} />}
      {sent && <p className={`small gateway-sent${sent.ok ? ' ok' : ''}`} role="status">{sent.text}</p>}
      {notes.map((note, at) => <p key={note} className="small muted row" style={{ margin: 0 }}>{note}{at === notes.length - 1 && undoButton}</p>)}
      {notes.length === 0 && undoButton && <p className="small row" style={{ margin: 0 }}>{undoButton}</p>}
      {state.phase === 'failed' && <SaveProblem text={GATEWAY_COPY.notSaved} />}
      {skipped.length > 0 && <ul className="gateway-problems" role="status">{skipped.slice(0, 20).map(problem =>
        <li key={problem}>{problem}</li>)}{skipped.length > 20 && <li>…and {skipped.length - 20} more</li>}</ul>}
      {problems.length > 0 && <ul className="gateway-problems" role="alert">{problems.slice(0, 20).map(problem =>
        <li key={problem}>{problem}</li>)}{problems.length > 20 && <li>…and {problems.length - 20} more</li>}</ul>}
      {stuck && <p className="small row" style={{ margin: 0 }}><button type="button" className="ghost" onClick={() => { setNotes([]); setSkipped([]); discard(); }}>
        <Undo2 size={14} />{GATEWAY_COPY.discard}</button>{(locked || state.phase === 'waiting') && <span className="muted">{GATEWAY_COPY.willSave}</span>}</p>}
    </div>}
    {rows && rows.length === 0 ? <p className="muted watch-empty">{GATEWAY_COPY.emptyPoints}</p> :
      <div className="table-scroll"><table className="gateway-points compact-rows"><thead><tr>
        {picked && !locked && <th className="pick-col"><input type="checkbox" aria-label="Tick all points shown"
          checked={shownKeys.length > 0 && shownKeys.every(key => picked.has(key))}
          onChange={() => setPicked(toggleShown(picked, shownKeys))} /></th>}<th>{GATEWAY_COPY.name}</th><th>Type</th>
        <th>{ADDRESS_LABEL[protocol]}</th><th>IED · path</th><th>Scale</th><th>Value now</th><th aria-label="Remove" /></tr></thead>
        <tbody>{shown.slice(first, first + PAGE).map(({ row, index }) => {
          const key = pointKey(row, index);
          return <Fragment key={key}><PointRow row={row} protocol={protocol} rows={rows ?? []} locked={locked} dirty={dirty}
            picking={!!picked && !locked} picked={!!picked?.has(key)} changing={changing === key} choices={choices}
            togglePicked={() => picked && setPicked(togglePick(picked, key))} toggleChange={() => setChanging(changing === key ? null : key)}
            update={change => update(index, change)} remove={() => removeRows((_, at) => at !== index)}
            setOwn={value => setOwn(index, row, value)} ask={choice => { setSent(null); setPending(choice); }} fail={fail} />
            {changing === key && !isOwn(row) && !locked && <SignalChange row={row} ieds={ieds} taken={takenFor(index, row.kind)}
              columns={picked ? 8 : 7} close={() => setChanging(null)} choose={(ied, ref) => {
                update(index, { ied: ied.id, ref, ied_name: ied.name, value_text: undefined, online: undefined });
                setChanging(null);
              }} />}</Fragment>;
        })}</tbody></table>
        {rows && rows.length > 0 && shown.length === 0 && <p className="muted watch-empty">{GATEWAY_COPY.emptyFilter}</p>}</div>}
    <Pager count={shown.length} page={page} size={PAGE} setPage={setPage} noun="points" />
  </>;
}
