import { useEffect, useState } from 'react';
import type { ModbusRange } from '../../../../../api/master';
import { masterListApi } from '../../../../../api/masterList';
import { DATA_TYPE_NAMES, TABLE_NAMES } from '../../../../../copy/master';

const WIDE = new Set(['int32', 'uint32', 'float32']);

/** The Modbus reads the engine adds for "My point list" (ADR-0110): neighbouring points of one table and
 *  data type are read in one request, capped per request. Read only here: change the list to change them. */
export function AutoRanges({ refresh }: { refresh: number }) {
  const [ranges, setRanges] = useState<ModbusRange[]>([]);
  useEffect(() => { masterListApi.list().then(list => setRanges(list.auto_ranges)).catch(() => setRanges([])); }, [refresh]);
  if (!ranges.length) return null;
  return <div className="guide-section auto-ranges">
    <strong className="small">Also read for My point list (automatic)</strong>
    <ul className="small muted">{ranges.map(range => {
      const last = range.start + range.count * (WIDE.has(range.data_type) ? 2 : 1) - 1;
      return <li key={`${range.table}-${range.start}-${range.data_type}`}>{TABLE_NAMES[range.table] ?? range.table} {range.start}
        {last > range.start ? `–${last}` : ''} · {range.count} × {DATA_TYPE_NAMES[range.data_type] ?? range.data_type}</li>;
    })}</ul>
  </div>;
}
