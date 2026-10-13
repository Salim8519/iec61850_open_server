import type { Signal } from '../../api/types';
import { SignalValueEditor } from '../../components/SignalValueEditor';

export function SignalTable({ signals, values, iedId, running, update }: {
  signals: Signal[]; values: Record<string, unknown>; iedId: string; running: boolean;
  update: (reference: string, value: unknown) => void;
}) {
  return <div className="table-scroll"><table>
    <thead><tr><th>Data reference</th><th>FC</th><th>Type</th><th>Value</th><th>Quality</th><th>Timestamp</th></tr></thead>
    <tbody>{signals.map(signal => {
      const quality = Number(values[`${iedId}:${signal.quality_ref}`] ?? signal.quality);
      const timestamp = Number(values[`${iedId}:${signal.timestamp_ref}`] ?? signal.timestamp);
      return <tr key={signal.ref}>
      <td className="reference-cell" title={signal.ref}>{signal.description && <div className="small">{signal.description}</div>}<code>{signal.ref}</code></td>
      <td><span className="badge">{signal.fc}</span></td><td className="muted">{signal.kind}</td>
      <td><SignalValueEditor signal={signal} value={values[`${iedId}:${signal.ref}`] ?? signal.value}
        disabled={!running} update={value => update(signal.ref, value)} /></td>
      <td className="muted">{!signal.quality_ref ? 'Not present' : quality === 0 ? 'Good' : `Flags ${quality}`}</td>
      <td className="muted" title={timestamp ? new Date(timestamp).toISOString() : undefined}>
        {timestamp ? new Date(timestamp).toLocaleTimeString() : 'Not set'}</td>
    </tr>; })}</tbody>
  </table>{!signals.length && <div className="empty">No matching signals.</div>}</div>;
}
