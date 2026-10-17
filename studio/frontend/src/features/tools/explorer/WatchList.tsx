import { Download, Eye, EyeOff, RefreshCw, Trash2, X } from 'lucide-react';
import type { ExplorerStatus, WatchItem } from '../../../api/tools';
import { explorerApi } from '../../../api/tools';
import { EXPLORER_COPY } from '../../../copy/tools';
import { Hint } from '../../../ui/Hint';
import { SegmentedControl } from '../../../ui/SegmentedControl';
import { timeOf, ValueText } from './ValueText';

const INTERVALS = [['500', '0.5 s'], ['1000', '1 s'], ['2000', '2 s'], ['5000', '5 s'], ['10000', '10 s']] as const;

/** Values the user chose to follow; the backend polls them, the page only shows the latest answer. */
export function WatchList({ status, run }: {
  status: ExplorerStatus; run: (task: () => Promise<ExplorerStatus>) => void;
}) {
  const watch = status.watch;
  const save = (items: WatchItem[], interval?: number) => run(() => explorerApi.watch(items, interval));
  const remove = (item: WatchItem) => save(watch.filter(entry => !(entry.ref === item.ref && entry.fc === item.fc)));
  const readOnce = () => run(async () => {
    await explorerApi.read(watch.map(({ ref, fc }) => ({ ref, fc })));
    return explorerApi.status();
  });
  return <div className="card explorer-watch">
    <div className="card-heading"><h3>{EXPLORER_COPY.watchTitle} ({watch.length})</h3>
      <div className="row">
        <span className="small muted">{EXPLORER_COPY.every}</span>
        <SegmentedControl label={EXPLORER_COPY.every} value={String(status.interval_ms)}
          options={INTERVALS.map(([value, label]) => ({ value, label }))}
          onChange={value => save(watch, Number(value))} />
        <button disabled={!watch.length} onClick={readOnce}><RefreshCw size={16} />{EXPLORER_COPY.readOnce}</button>
        <button className={status.polling ? '' : 'primary'} disabled={!watch.length}
          onClick={() => run(() => explorerApi.polling(!status.polling))}>
          {status.polling ? <><EyeOff size={16} />{EXPLORER_COPY.stop}</> : <><Eye size={16} />{EXPLORER_COPY.start}</>}</button>
        <a className={`button${watch.length ? '' : ' disabled'}`} href={explorerApi.csvUrl} download
          aria-disabled={!watch.length}><Download size={16} />{EXPLORER_COPY.exportCsv}</a>
        <button className="ghost" disabled={!watch.length} onClick={() => save([])}><Trash2 size={16} />{EXPLORER_COPY.clear}</button>
      </div></div>
    <div className="guide-section"><Hint id="tools.explorer.watch" compact /></div>
    {watch.length === 0 ? <p className="muted watch-empty">{EXPLORER_COPY.watchEmpty}</p>
      : <div className="table-scroll"><table><thead><tr><th>Reference</th><th>FC</th><th>Value</th><th>Read at</th>
        <th aria-label="Remove" /></tr></thead>
        <tbody>{watch.map(item => <tr key={`${item.ref}|${item.fc}`}>
          <td className="mono reference-cell" title={item.ref}>{item.ref}</td><td><span className="fc-badge">{item.fc}</span></td>
          <td><ValueText text={item.text} status={item.status} /></td><td className="small muted">{timeOf(item.at)}</td>
          <td><button className="ghost icon-button" onClick={() => remove(item)} aria-label={`Stop watching ${item.ref}`}
            title="Remove from watch list"><X size={15} /></button></td></tr>)}</tbody></table></div>}
    {status.polling && <p className="small muted watch-foot" aria-live="polite">Watching · {status.reads} reads so far</p>}
  </div>;
}
