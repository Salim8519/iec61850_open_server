import { ChevronDown, Copy, Download, ListPlus, Plus, Upload } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import type { MasterCommand, MasterStatus } from '../../../../../api/master';
import { masterListApi, type ComparedRow, type ListPoint } from '../../../../../api/masterList';
import { GATEWAY_COPY } from '../../../../../copy/gateway';
import { LIST_COPY } from '../../../../../copy/masterList';
import { useListFilter } from '../../../../../state/useListFilter';
import { FilterBar } from '../../../../../ui/FilterBar';
import { HelpTip } from '../../../../../ui/HelpTip';
import { Menu } from '../../../../../ui/Menu';
import { SaveProblem, SaveStatus } from '../../../../../ui/SaveStatus';
import { saveFile } from '../../saveFile';
import { MasterCommandForm } from '../MasterCommandForm';
import { ListAddDialog, type ListSource } from './ListAddDialog';
import { ListTable } from './ListTable';
import { appendPoints, changePoint, draftOf, EXTRA, extraMatches, LIST_FILTER, listRowMatches, removePoints, stateCounts } from './listRows';
import { NotInList } from './NotInList';
import { useMasterList } from './useMasterList';
import './master-list.css';

/** "My point list" of the SCADA master (ADR-0110): the addresses and types the master expects, compared by
 *  the engine with what the device sends (Received, Missing, Wrong type), what the device sends beyond the
 *  list (Not in your list, one click to add), one-click commands and the optional Check against IED. */
export function MasterList({ status, fail }: { status: MasterStatus; fail: (message: string) => void }) {
  const { list, comparison, edit, undo, restore, problem, state, setCheck } = useMasterList(status.protocol, fail);
  const filter = useListFilter('master-list', LIST_FILTER);
  const [adding, setAdding] = useState<ListSource | null>(null);
  const [note, setNote] = useState('');
  const [draft, setDraft] = useState<MasterCommand | null>(null);
  const messages = useRef<HTMLDivElement>(null);
  // Send opens the confirmation above the list: bring it into view (the list may be scrolled far down).
  useEffect(() => { if (draft) messages.current?.scrollIntoView({ block: 'nearest', behavior: 'smooth' }); }, [draft]);
  const types = useMemo(() => new Map((list?.types ?? []).map(type => [type.kind, type])), [list]);
  const rows = useMemo<ComparedRow[]>(() => {
    const found = new Map((comparison?.rows ?? []).map(row => [row.id, row]));
    return (list?.points ?? []).map(point => {
      const known = found.get(point.id);
      return known ? { ...known, point } : { id: point.id, point, state: 'waiting', reason: '', command: null };
    });
  }, [list, comparison]);
  if (!list) return <p className="muted watch-empty">Loading the point list…</p>;
  const points = list.points;
  const added = (more: ListPoint[], words: string[] = []) => {
    edit(appendPoints(points, more), points);
    setAdding(null); setNote([LIST_COPY.added(more.length), ...words].join(' '));
  };
  const copy = () => masterListApi.copyGateway().then(plan => {
    if (plan.points.length) return added(plan.points, plan.notes ?? []);
    setNote([plan.gateway_points ? LIST_COPY.copyNothingNew : LIST_COPY.copyNone, ...(plan.notes ?? []).slice(1)].join(' '));
  }).catch(error => fail(error.message));
  const fromReceived = (keys: string[]) => masterListApi.fromReceived(keys).then(plan => plan.ok ? added(plan.points)
    : setNote(plan.rows.find(row => row.problem)?.problem ?? '')).catch(error => fail(error.message));
  const remove = (ids: string[]) => { edit(removePoints(points, new Set(ids)), points); setNote(LIST_COPY.removed(ids.length)); };
  const extras = filter.values.state === EXTRA;
  const shownRows = rows.filter(row => listRowMatches(row, filter.values, types.get(row.point.type)));
  const shownExtras = (comparison?.not_in_list ?? []).filter(row => extraMatches(row, filter.values));
  return <>
    <div className="panel-toolbar master-toolbar">
      <button className="primary" onClick={() => setAdding('one')}><Plus size={16} />{LIST_COPY.add}</button>
      <button onClick={copy} title={LIST_COPY.copyHint}><Copy size={16} />{LIST_COPY.copy}</button>
      <Menu label={LIST_COPY.moreLabel} align="left" trigger={<>{LIST_COPY.more}<ChevronDown size={15} aria-hidden /></>} items={[
        { label: LIST_COPY.exportCsv, icon: <Download size={16} />, onSelect: () => saveFile(masterListApi.csvUrl()) },
        { label: LIST_COPY.importCsv, icon: <Upload size={16} />, onSelect: () => setAdding('paste') },
        { label: LIST_COPY.template, icon: <ListPlus size={16} />, onSelect: () => saveFile(masterListApi.templateUrl()) }]} />
      <HelpTip text={LIST_COPY.hint} label="What is this list?" />
      <span className="gateway-spacer" />
      <label className="row gateway-check" title={LIST_COPY.checkHint}><input type="checkbox" checked={list.check_ied}
        onChange={event => setCheck(event.target.checked)} />{LIST_COPY.checkIed} <span className="small muted">(test aid)</span></label>
      <SaveStatus state={state} />
      <strong className="small">{LIST_COPY.count(points.length)}</strong>
    </div>
    <FilterBar label="Filter my point list" spec={LIST_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={extras ? shownExtras.length : shownRows.length} total={extras ? comparison?.not_in_list.length ?? 0 : rows.length}
      counts={stateCounts(comparison)} />
    {(note || problem || undo || draft) && <div ref={messages} className="guide-section panel-messages">
      {draft && <MasterCommandForm key={`${draft.kind}-${draft.address}`} draft={draft} host={status.host} protocol={status.protocol}
        review close={() => setDraft(null)} fail={fail} />}
      <SaveProblem text={problem} />
      {(note || undo) && <p className="small muted row" style={{ margin: 0 }}>{note}
        {undo && <button className="ghost" onClick={() => { restore(); setNote(''); }}>{LIST_COPY.undo}</button>}</p>}
    </div>}
    <div className="table-scroll">
      {extras ? <NotInList rows={shownExtras} protocol={status.protocol} add={fromReceived} />
        : points.length === 0 ? <div className="guide-empty"><h3>{LIST_COPY.empty}</h3><p>{LIST_COPY.emptyHint}</p></div>
        : <ListTable rows={shownRows} protocol={status.protocol} types={types} check={list.check_ied} linkOpen={status.link_open}
          change={(id, patch) => edit(changePoint(points, id, patch))} remove={id => remove([id])}
          send={row => setDraft(draftOf(row))} />}
      {!extras && points.length > 0 && shownRows.length === 0 && <p className="muted watch-empty">{GATEWAY_COPY.emptyFilter}</p>}
    </div>
    {adding && <ListAddDialog protocol={status.protocol} types={list.types} start={adding} add={more => added(more)}
      close={() => setAdding(null)} />}
  </>;
}
