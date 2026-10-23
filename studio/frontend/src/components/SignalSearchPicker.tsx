import { useEffect, useState } from 'react';
import { Search } from 'lucide-react';
import type { Ied, Signal, SignalBinding } from '../api/types';
import type { Kind } from '../points/pointKind';
import { Spinner } from '../ui/Spinner';
import { ALL_KINDS } from '../ui/kindFilter';
import { PICKER_COPY } from '../copy/signalPicker';
import { initialView, rememberView, type PickerView } from './pickerMemory';
import { SIGNAL_PAGE, signalKey } from './signalQuery';
import { KindSelect } from './KindSelect';
import { HelperToggle } from './signalPicker/HelperToggle';
import { KindChips } from './signalPicker/KindChips';
import { ObjectSelect } from './signalPicker/ObjectSelect';
import { isSmart, kindOfPicked, type PickedSignal } from './signalPicker/pickerAnswer';
import { preferredKind, takesHelpers } from './signalPicker/preferredKind';
import { SignalPickRow } from './signalPicker/SignalPickRow';
import { SuggestionBlock } from './signalPicker/SuggestionBlock';
import { useSignalSearch } from './signalPicker/useSignalSearch';
import './signal-search-picker.css';

type Props = {
  ieds: Ied[]; fixedIedId?: string; allowedKinds?: string[]; allowedFc?: string;
  writableOnly?: boolean; exclude?: Set<string>; disabled?: boolean; chooseLabel?: string;
  /** Remembers IED, search, report, type, object and page under this name, so reopening continues where the user was. */
  memoryKey?: string;
  /** The point chosen now ("iedId:ref"): its row says Chosen, the others offer to replace it. */
  chosen?: SignalBinding;
  choose: (binding: SignalBinding, signal: Signal) => void;
  /** Adds every not-yet-added signal of the page in one click (measurement lists, ADR-0090). */
  chooseAll?: (items: { binding: SignalBinding; signal: Signal }[]) => void;
  /** The component's name: points that mention it are suggested first (ADR-0113). */
  near?: string;
  /** What this slot is, in the note that says why a type is chosen ("Open contact"). */
  slot?: string;
  /** A command slot's own kind (SC, DC or SE) when the accepted types cannot tell it. */
  command?: Kind;
};

/** Search-first signal chooser: pick an IED once, then search, narrow by type and object, and one-click add.
 *  With the Go engine (ADR-0113) the slot's type is chosen, the engine counts types and objects and suggests
 *  the points that mention the component; an older engine gets the list as before. */
export function SignalSearchPicker({ ieds, fixedIedId, allowedKinds, allowedFc, writableOnly, exclude, disabled,
  chooseLabel = 'Choose', memoryKey, chosen, choose, chooseAll, near, slot, command }: Props) {
  const onlyIed = ieds.length === 1 ? ieds[0].id : '';
  const [view, setView] = useState<PickerView>(() =>
    initialView(memoryKey, fixedIedId || chosen?.ied_id, onlyIed));
  const iedId = fixedIedId || view.iedId || onlyIed;
  const { query, reportRef, offset } = view;
  const preferred = preferredKind({ kinds: allowedKinds, fc: allowedFc, command }) ?? ALL_KINDS;
  // An engine without picker support (the frozen Python engine) answers without counts: then the list
  // starts on All types as before. A type chosen by the user is kept for this slot; another slot under
  // the same name opens on its own type.
  const [legacy, setLegacy] = useState(false);
  const remembered = view.pointKind !== undefined && (view.kindFor ?? ALL_KINDS) === preferred;
  const pointKind = remembered ? view.pointKind! : legacy ? ALL_KINDS : preferred;
  const ln = view.ln ?? '';
  const name = near?.trim() ?? '';
  const ied = ieds.find(item => item.id === iedId);
  const { answer, loading, error } = useSignalSearch(iedId, { q: query, reportRef, offset, kinds: allowedKinds,
    fc: allowedFc, writableOnly, pointKind, smart: true, helpers: view.helpers, ln, near: name });
  const smart = isSmart(answer);
  useEffect(() => { if (answer) setLegacy(!smart); }, [answer, smart]);
  const items = answer?.items ?? [];
  const total = answer?.total ?? 0;
  // A new search, report, type or IED starts again at the first page; paging keeps the rest.
  const change = (next: Partial<PickerView>) => setView(current => ({ ...current, offset: 0, ...next }));
  useEffect(() => { rememberView(memoryKey, { ...view, iedId }); }, [memoryKey, view, iedId]);
  const chosenKey = chosen ? signalKey(chosen.ied_id, chosen.ref) : '';
  const binding = (signal: Signal): SignalBinding => ({ ied_id: iedId,
    report_ref: reportRef === '__all__' ? null : reportRef, ref: signal.ref });
  const open = items.filter(signal => !exclude?.has(signalKey(iedId, signal.ref)));
  const row = (signal: PickedSignal, note?: string) => {
    const key = signalKey(iedId, signal.ref);
    return <SignalPickRow key={`${note ? 'suggested:' : ''}${signal.ref}`} signal={signal}
      kind={kindOfPicked(signal, ied?.controls)} iedName={ied?.name} chosen={key === chosenKey}
      taken={!!exclude?.has(key)} replacing={!!chosenKey} disabled={!!disabled || loading} chooseLabel={chooseLabel}
      note={note} choose={() => choose(binding(signal), signal)} />;
  };
  const why = smart && pointKind === preferred && preferred !== ALL_KINDS ? PICKER_COPY.why(preferred as Kind, slot) : undefined;
  return <div className="signal-search-picker">
    {!fixedIedId && <label>IED<select value={iedId} disabled={disabled}
      onChange={e => change({ iedId: e.target.value, query: '', reportRef: '__all__', ln: '' })}>
      <option value="">Choose an IED</option>
      {ieds.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}
    </select></label>}
    {ied && <>
      <div className="search-field signal-search-query"><Search size={14} />
        <input aria-label={PICKER_COPY.searchLabel} placeholder={PICKER_COPY.search} value={query}
          disabled={disabled} onChange={e => change({ query: e.target.value })} /></div>
      {smart && <KindChips counts={answer?.kind_counts ?? {}} value={pointKind} why={why} disabled={disabled}
        onChange={next => change({ pointKind: next, kindFor: preferred })} />}
      <div className="signal-search-filters">
        {smart && <ObjectSelect counts={answer?.ln_counts ?? {}} value={ln} disabled={disabled}
          total={Object.values(answer?.ln_counts ?? {}).reduce((sum, count) => sum + count, 0)}
          onChange={next => change({ ln: next })} />}
        <select aria-label="Report" value={reportRef} disabled={disabled}
          onChange={e => change({ reportRef: e.target.value })}>
          <option value="__all__">All reports</option>
          {ied.reports.map(report => <option key={report.ref} value={report.ref}>{report.ref}</option>)}
        </select>
        {answer && !smart && <KindSelect value={pointKind} disabled={disabled}
          onChange={next => change({ pointKind: next, kindFor: preferred })} />}
      </div>
      {smart && takesHelpers(allowedKinds) && <HelperToggle shown={!!view.helpers} disabled={disabled}
        onChange={shown => change({ helpers: shown })} />}
    </>}
    {error && <p role="alert">{error}</p>}
    {ied && name && <SuggestionBlock name={name} items={answer?.suggested ?? []} row={row} />}
    {ied && <><div className="signal-search-list" aria-busy={loading}>
      {loading && !items.length && <Spinner label="Loading signals…" />}
      {items.map(signal => row(signal))}
      {!loading && answer && !items.length && <p className="small muted">
        {smart && pointKind !== ALL_KINDS ? PICKER_COPY.noneOfKind(pointKind as Kind) : PICKER_COPY.none}</p>}
    </div>
    {chooseAll && open.length > 1 && <button type="button" className="signal-search-all" disabled={disabled || loading}
      title="Adds every signal listed above that is not added yet"
      onClick={() => chooseAll(open.map(signal => ({ binding: binding(signal), signal })))}>
      {`${chooseLabel} all ${open.length} shown`}</button>}
    <div className="row small muted signal-search-pages">
      <span>{total ? offset + 1 : 0}–{Math.min(offset + SIGNAL_PAGE, total)} of {total}</span>
      {loading && !!items.length && <Spinner inline size={14} label="Loading…" />}
      <span style={{ flex: 1 }} />
      <button type="button" className="ghost" disabled={loading || offset === 0}
        onClick={() => setView({ ...view, offset: offset - SIGNAL_PAGE })}>Previous</button>
      <button type="button" className="ghost" disabled={loading || offset + SIGNAL_PAGE >= total}
        onClick={() => setView({ ...view, offset: offset + SIGNAL_PAGE })}>Next</button></div></>}
  </div>;
}
