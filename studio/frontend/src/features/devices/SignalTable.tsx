import { memo } from 'react';
import { ListPlus } from 'lucide-react';
import type { Signal } from '../../api/types';
import { ValueControl } from '../test/ValueControl';
import { Spinner } from '../../ui/Spinner';
import { useSignalValue } from '../../state/useLive';

type Perform = (task: () => Promise<unknown>) => Promise<void>;
type RowProps = { signal: Signal; iedId: string; running: boolean; perform: Perform; addTest?: (signal: Signal) => void };

/** One data point; it reads its own live value and quality, so a value burst repaints only the
 * rows whose value changed (speed lane F). */
const SignalRow = memo(function SignalRow({ signal, iedId, running, perform, addTest }: RowProps) {
  const live = useSignalValue(`${iedId}:${signal.ref}`);
  const liveQuality = useSignalValue(signal.quality_ref ? `${iedId}:${signal.quality_ref}` : null);
  const quality = Number(liveQuality ?? signal.quality);
  const writable = signal.writable && signal.fc !== 'CF';
  return <tr>
    <td className="reference-cell" title={signal.ref}>{signal.description && <div className="small">{signal.description}</div>}
      <code>{signal.ref}</code></td>
    <td><span className="badge">{signal.fc}</span></td><td className="muted small">{signal.kind}</td>
    <td><ValueControl point={{ ied_id: iedId, ref: signal.ref }} signal={signal} running={running} perform={perform}
      value={live ?? signal.value} /></td>
    <td className="muted small">{!signal.quality_ref ? '—' : quality === 0 ? 'Good' : `Flags ${quality}`}</td>
    <td>{addTest && writable && !running && <button className="icon ghost" title="Add to Test points"
      aria-label={`Add ${signal.ref} to test points`} onClick={() => addTest(signal)}><ListPlus size={17} /></button>}</td>
  </tr>;
});

/** The device's data points; values change through the shared ValueControl (UI-06). */
export function SignalTable({ signals, iedId, running, perform, addTest, loading = false }: {
  signals: Signal[]; iedId: string; running: boolean; loading?: boolean;
  perform: Perform; addTest?: (signal: Signal) => void;
}) {
  return <div className="table-scroll"><table>
    <thead><tr><th>Data point</th><th>FC</th><th>Type</th><th>Value</th><th>Quality</th><th /></tr></thead>
    <tbody>{signals.map(signal => <SignalRow key={signal.ref} signal={signal} iedId={iedId} running={running}
      perform={perform} addTest={addTest} />)}</tbody>
  </table>{!signals.length && (loading ? <div className="empty"><Spinner label="Loading data points…" /></div>
    : <div className="empty">No data points match your search.</div>)}</div>;
}
