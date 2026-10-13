import { X } from 'lucide-react';
import type { StudioEvent } from '../../api/types';
import { eventDetails } from './eventDetails';

/** Activity slide-over: what the engine did, newest first. */
export function EventLog({ events, session, close }: {
  events: StudioEvent[]; session?: string; close: () => void;
}) {
  return <>
    <div className="scrim" onClick={close} />
    <aside className="activity" aria-label="Activity">
      <header><div><h2>Activity</h2>{session && <span className="small muted">Session {session}</span>}</div>
        <button className="icon ghost" aria-label="Close activity" onClick={close}><X size={18} /></button></header>
      <div className="activity-body"><table><thead><tr><th>Time</th><th>Event</th><th>Details</th></tr></thead>
        <tbody>{[...events].reverse().map(event => <tr key={event.id}>
          <td className="mono" style={{ width: 96 }}>{new Date(event.timestamp * 1000).toLocaleTimeString()}</td>
          <td className="event-kind">{event.kind.replaceAll('_', ' ')}</td>
          <td className="mono">{eventDetails(event)}</td>
        </tr>)}</tbody></table>
        {!events.length && <div className="empty">Nothing yet. Press Run to start the IEDs; everything they do appears here.</div>}
      </div>
    </aside>
  </>;
}
