import { Trash2 } from 'lucide-react';

/** The bulk delete endpoint reviews at most this many IEDs at once (projects/bulk_removal.py). */
export const MAX_BULK = 20;

/** Footer of the Devices list while choosing several IEDs: count, select all, cancel, delete. */
export function DeviceSelectBar({ count, shown, selectShown, cancel, remove }: {
  count: number; shown: number; selectShown: () => void; cancel: () => void; remove: () => void;
}) {
  const tooMany = count > MAX_BULK;
  return <div className="side-foot device-select-bar">
    <div className="row" style={{ gap: 6 }}>
      <strong className="small" style={{ whiteSpace: 'nowrap' }}>{count} selected</strong>
      <span style={{ flex: 1 }} />
      {shown > 0 && <button className="ghost small-button" onClick={selectShown}>All {shown}</button>}
      <button className="ghost small-button" onClick={cancel}>Cancel</button>
    </div>
    {tooMany && <p className="side-line">Delete up to {MAX_BULK} IEDs at a time.</p>}
    <button className="danger" disabled={!count || tooMany} onClick={remove}>
      <Trash2 size={16} />{count === 1 ? 'Delete 1 IED' : `Delete ${count} IEDs`}</button>
  </div>;
}
