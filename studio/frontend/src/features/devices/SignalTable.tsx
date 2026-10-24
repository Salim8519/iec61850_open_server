import { memo } from 'react';
import { ListPlus } from 'lucide-react';
import type { Signal } from '../../api/types';
import { TypeCell } from '../../components/TypeCell';
import type { Kind } from '../../points/pointKind';
import { kindRow } from '../../ui/KindBadge';
import { ValueControl } from '../test/ValueControl';
import { Spinner } from '../../ui/Spinner';
import { useSignalValue } from '../../state/useLive';
import type { DeadbandPoint } from './useDeadbands';

type Perform = (task: () => Promise<unknown>) => Promise<void>;
type RowProps = { signal: Signal; kind: Kind; iedId: string; running: boolean; perform: Perform; addTest?: (signal: Signal) => void;
  deadband?: DeadbandPoint };

/** One data point; it reads its own live value and quality, so a value burst repaints only the
 * rows whose value changed (speed lane F). The kind is decided by the table and passed in, so a
 * repaint never recomputes it. */
const SignalRow = memo(function SignalRow({ signal, kind, iedId, running, perform, addTest, deadband }: RowProps) {
  const live = useSignalValue(`${iedId}:${signal.ref}`);
  const liveQuality = useSignalValue(signal.quality_ref ? `${iedId}:${signal.quality_ref}` : null);
  const quality = Number(liveQuality ?? signal.quality);
  const writable = signal.writable && signal.fc !== 'CF';
  return <tr {...kindRow(kind)}>
    <TypeCell kind={kind} raw={signal.kind} fc={signal.fc} />
    <td className="reference-cell point-cell" title={signal.ref}>
      {signal.description && <span className="point-desc">{signal.description}</span>}<code>{signal.ref}</code>
      {deadband && <span className="point-deadband small muted" title={deadband.note}>
        db {deadband.db} from the SCD · {deadband.applies ? 'applied' : 'not applied'}</span>}</td>
    <td><ValueControl point={{ ied_id: iedId, ref: signal.ref }} signal={signal} running={running} perform={perform}
      value={live ?? signal.value} /></td>
    <td className="muted small">{!signal.quality_ref ? '—' : quality === 0 ? 'Good' : `Flags ${quality}`}</td>
    <td>{addTest && writable && !running && <button className="icon ghost" title="Add to Test points"
      aria-label={`Add ${signal.ref} to test points`} onClick={() => addTest(signal)}><ListPlus size={17} /></button>}</td>
  </tr>;
});

/** The device's data points; values change through the shared ValueControl (UI-06). */
export function SignalTable({ signals, kindOf, iedId, running, perform, addTest, loading = false, deadbands }: {
  signals: Signal[]; kindOf: (signal: Signal) => Kind; iedId: string; running: boolean; loading?: boolean;
  perform: Perform; addTest?: (signal: Signal) => void; deadbands?: Map<string, DeadbandPoint>;
}) {
  return <div className="table-scroll"><table className="compact-rows">
    <thead><tr><th title="Hover a badge for its full name and the exact data type">Type</th><th>Data point</th><th>Value</th><th>Quality</th><th /></tr></thead>
    <tbody>{signals.map(signal => <SignalRow key={signal.ref} signal={signal} kind={kindOf(signal)} iedId={iedId}
      running={running} perform={perform} addTest={addTest} deadband={deadbands?.get(signal.ref)} />)}</tbody>
  </table>{!signals.length && (loading ? <div className="empty"><Spinner label="Loading data points…" /></div>
    : <div className="empty">No data points match your search.</div>)}</div>;
}
