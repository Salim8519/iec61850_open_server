import type { StudioEvent } from '../../api/types';
import { eventDetails } from './eventDetails';

export function EventLog({ events }: { events: StudioEvent[] }) {
  return <div className="event-drawer"><table><thead><tr><th>Time</th><th>Event</th><th>Details</th></tr></thead>
    <tbody>{[...events].reverse().map(event => <tr key={event.id}>
      <td className="mono" style={{ width: 120 }}>{new Date(event.timestamp * 1000).toLocaleTimeString()}</td>
      <td className="event-kind">{event.kind.replaceAll('_', ' ')}</td>
      <td className="mono">{eventDetails(event)}</td>
    </tr>)}</tbody></table>{!events.length && <div className="empty">Run an IED to begin recording events.</div>}</div>;
}
