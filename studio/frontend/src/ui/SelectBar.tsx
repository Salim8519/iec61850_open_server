import { Trash2 } from 'lucide-react';
import { plural } from './pickRows';

/** Bar above a table in Select mode: how many rows are ticked, tick all shown, cancel, delete. */
export function SelectBar({ count, shown, noun, selectShown, cancel, remove }: {
  count: number; shown: number; noun: string; selectShown: () => void; cancel: () => void; remove: () => void;
}) {
  return <div className="select-bar" role="region" aria-label={`Choose ${noun}s to delete`}>
    <strong className="small" role="status">{count.toLocaleString('en-US')} selected</strong>
    <span className="small muted">Tick rows, or use the search and filters and press All shown.</span>
    <span className="select-bar-gap" />
    {shown > 0 && <button type="button" className="ghost" onClick={selectShown}>All {shown.toLocaleString('en-US')} shown</button>}
    <button type="button" className="ghost" onClick={cancel}>Cancel</button>
    <button type="button" className="danger" disabled={!count} onClick={remove}>
      <Trash2 size={16} />Delete {plural(count, noun)}</button>
  </div>;
}
