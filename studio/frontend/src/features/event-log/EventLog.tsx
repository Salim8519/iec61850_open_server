import { useCallback, useRef } from 'react';
import { X } from 'lucide-react';
import { liveData } from '../../state/liveData';
import { useEventLogVersion } from '../../state/useLive';
import { SpacerRow, useVirtualRows } from '../../ui/VirtualRows';

const ROW_HEIGHT = 34;

/** Activity slide-over: what the engine did, newest first. Reads the event log store directly and
 * draws only the rows on screen (speed lane F); time and kind words are made once per event. */
export function EventLog({ session, close }: { session?: string; close: () => void }) {
  const log = liveData().log;
  useEventLogVersion();
  const size = log.size();
  const box = useRef<HTMLDivElement>(null);
  const keyOf = useCallback((index: number) => log.at(index)?.id ?? index, [log]);
  const { range, spacer } = useVirtualRows(box, size, keyOf, ROW_HEIGHT);
  const rows = [];
  for (let index = range.start; index < Math.min(range.end, size); index++) {
    const entry = log.at(index);
    if (!entry) continue;
    rows.push(<tr key={entry.id} data-vrow={entry.id}>
      <td className="mono" style={{ width: 96, whiteSpace: 'nowrap' }}>{entry.time}</td>
      <td className="event-kind">{entry.label}</td>
      <td className="mono">{log.details(entry)}</td>
    </tr>);
  }
  return <>
    <div className="scrim" onClick={close} />
    <aside className="activity" aria-label="Activity">
      <header><div><h2>Activity</h2>{session && <span className="small muted">Session {session}</span>}</div>
        <button className="icon ghost" aria-label="Close activity" onClick={close}><X size={18} /></button></header>
      <div className="activity-body" ref={box}><table><thead><tr><th>Time</th><th>Event</th><th>Details</th></tr></thead>
        <tbody><SpacerRow height={range.before} columns={3} spacerRef={spacer} />{rows}
          <SpacerRow height={range.after} columns={3} /></tbody></table>
        {!size && <div className="empty">Nothing yet. Press Run to start the IEDs; everything they do appears here.</div>}
      </div>
    </aside>
  </>;
}
