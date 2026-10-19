import { ListPlus } from 'lucide-react';
import type { Signal } from '../../api/types';
import { ValueControl } from '../test/ValueControl';
import { Spinner } from '../../ui/Spinner';

/** The device's data points; values change through the shared ValueControl (UI-06). */
export function SignalTable({ signals, values, iedId, running, perform, addTest, loading = false }: {
  signals: Signal[]; values: Record<string, unknown>; iedId: string; running: boolean; loading?: boolean;
  perform: (task: () => Promise<unknown>) => Promise<void>; addTest?: (signal: Signal) => void;
}) {
  return <div className="table-scroll"><table>
    <thead><tr><th>Data point</th><th>FC</th><th>Type</th><th>Value</th><th>Quality</th><th /></tr></thead>
    <tbody>{signals.map(signal => {
      const quality = Number(values[`${iedId}:${signal.quality_ref}`] ?? signal.quality);
      const writable = signal.writable && signal.fc !== 'CF';
      return <tr key={signal.ref}>
        <td className="reference-cell" title={signal.ref}>{signal.description && <div className="small">{signal.description}</div>}
          <code>{signal.ref}</code></td>
        <td><span className="badge">{signal.fc}</span></td><td className="muted small">{signal.kind}</td>
        <td><ValueControl point={{ ied_id: iedId, ref: signal.ref }} signal={signal} running={running} perform={perform}
          value={values[`${iedId}:${signal.ref}`] ?? signal.value} /></td>
        <td className="muted small">{!signal.quality_ref ? '—' : quality === 0 ? 'Good' : `Flags ${quality}`}</td>
        <td>{addTest && writable && !running && <button className="icon ghost" title="Add to Test points"
          aria-label={`Add ${signal.ref} to test points`} onClick={() => addTest(signal)}><ListPlus size={17} /></button>}</td>
      </tr>;
    })}</tbody>
  </table>{!signals.length && (loading ? <div className="empty"><Spinner label="Loading data points…" /></div>
    : <div className="empty">No data points match your search.</div>)}</div>;
}
