import { Download, Eye, EyeOff, RefreshCw, Trash2, X } from 'lucide-react';
import { useCallback, useRef } from 'react';
import type { ExplorerStatus, WatchItem } from '../../../api/tools';
import { explorerApi } from '../../../api/tools';
import { EXPLORER_COPY } from '../../../copy/tools';
import { hint } from '../../../copy/hints';
import { useListFilter } from '../../../state/useListFilter';
import { FilterBar } from '../../../ui/FilterBar';
import { HelpTip } from '../../../ui/HelpTip';
import { SpacerRow, useVirtualRows } from '../../../ui/VirtualRows';
import { trackChanges, WATCH_FILTER, watchKey, watchMatches } from './explorerFilters';
import { timeOf, ValueText } from './ValueText';

const ROW_HEIGHT = 37;
const INTERVALS = [['500', '0.5 s'], ['1000', '1 s'], ['2000', '2 s'], ['5000', '5 s'], ['10000', '10 s']] as const;

/** Values the user chose to follow, pinned at the bottom of the right pane with its own scroll.
 *  The backend polls them; the page only shows the latest answer. */
export function WatchList({ status, run }: {
  status: ExplorerStatus; run: (task: () => Promise<ExplorerStatus>) => void;
}) {
  const watch = status.watch;
  const filter = useListFilter('explorer-watch', WATCH_FILTER);
  const changes = useRef(new Map<string, { text: string; at: number }>());
  const now = Date.now();
  changes.current = trackChanges(changes.current, watch, now);
  const save = (items: WatchItem[], interval?: number) => run(() => explorerApi.watch(items, interval));
  const remove = (item: WatchItem) => save(watch.filter(entry => !(entry.ref === item.ref && entry.fc === item.fc)));
  const readOnce = () => run(async () => {
    await explorerApi.read(watch.map(({ ref, fc }) => ({ ref, fc })));
    return explorerApi.status();
  });
  const shown = watch.filter(item => watchMatches(item, filter.values, changes.current.get(watchKey(item))?.at ?? 0, now));
  // Long watch lists draw only the rows on screen (speed lane F).
  const box = useRef<HTMLDivElement>(null);
  const shownRef = useRef(shown);
  shownRef.current = shown;
  const keyOf = useCallback((index: number) => { const item = shownRef.current[index]; return item ? watchKey(item) : index; }, []);
  const { range, spacer } = useVirtualRows(box, shown.length, keyOf, ROW_HEIGHT);
  return <div className="card fill-panel explorer-watch">
    <div className="panel-head"><h3>{EXPLORER_COPY.watchTitle} ({watch.length})</h3>
      <HelpTip text={hint('tools.explorer.watch')} label="What is the watch list?" />
      <span className="gateway-spacer" />
      <select className="watch-interval" aria-label={EXPLORER_COPY.every} value={String(status.interval_ms)}
        onChange={event => save(watch, Number(event.target.value))}>
        {INTERVALS.map(([value, label]) => <option key={value} value={value}>{EXPLORER_COPY.everyShort} {label}</option>)}</select>
      <button disabled={!watch.length} onClick={readOnce}><RefreshCw size={16} />{EXPLORER_COPY.readOnce}</button>
      <button className={status.polling ? '' : 'primary'} disabled={!watch.length}
        onClick={() => run(() => explorerApi.polling(!status.polling))}>
        {status.polling ? <><EyeOff size={16} />{EXPLORER_COPY.stop}</> : <><Eye size={16} />{EXPLORER_COPY.start}</>}</button>
      <a className={`button${watch.length ? '' : ' disabled'}`} href={explorerApi.csvUrl} download
        aria-disabled={!watch.length}><Download size={16} />{EXPLORER_COPY.exportCsv}</a>
      <button className="ghost icon-button" disabled={!watch.length} onClick={() => save([])} aria-label={EXPLORER_COPY.clearWatch}
        title={EXPLORER_COPY.clearWatch}><Trash2 size={16} /></button>
    </div>
    <FilterBar label="Filter watched values" spec={WATCH_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={watch.length}
      extra={status.polling ? <span className="small muted" aria-live="polite">Watching · {status.reads} reads</span> : undefined} />
    <div className="panel-body">
      {watch.length === 0 ? <p className="muted watch-empty">{EXPLORER_COPY.watchEmpty}</p>
        : <div className="table-scroll" ref={box}><table><thead><tr><th>Reference</th><th>FC</th><th>Value</th><th>Read at</th>
          <th aria-label="Remove" /></tr></thead>
          <tbody><SpacerRow height={range.before} columns={5} spacerRef={spacer} />
          {shown.slice(range.start, range.end).map(item => <tr key={watchKey(item)} data-vrow={watchKey(item)}>
            <td className="mono reference-cell" title={item.ref}>{item.ref}</td><td><span className="fc-badge">{item.fc}</span></td>
            <td><ValueText text={item.text} status={item.status} /></td><td className="small muted">{timeOf(item.at)}</td>
            <td><button className="ghost icon-button" onClick={() => remove(item)} aria-label={`Stop watching ${item.ref}`}
              title="Remove from watch list"><X size={15} /></button></td></tr>)}
          <SpacerRow height={range.after} columns={5} /></tbody></table>
          {shown.length === 0 && <p className="muted watch-empty">{EXPLORER_COPY.noMatch}</p>}</div>}
    </div>
  </div>;
}
