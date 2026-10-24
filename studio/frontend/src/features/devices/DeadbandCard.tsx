import { api } from '../../api/client';
import { countOf } from '../../copy/count';
import type { Ied } from '../../api/types';
import { Disclosure } from '../../ui/Disclosure';
import { Hint } from '../../ui/Hint';
import type { Deadbands } from './useDeadbands';

/** The "Use SCD deadbands" switch of a simulated IED and what each measurement's db does (ADR-0118). */
export function DeadbandCard({ ied, deadbands, perform }: { ied: Ied; deadbands?: Deadbands;
  perform: (task: () => Promise<unknown>) => Promise<void> }) {
  if (ied.role !== 'simulated' || !deadbands?.points.length) return null;
  const count = deadbands.points.length;
  return <div className="guide-section"><h3 style={{ margin: 0 }}>Deadbands from the SCD</h3>
    <Hint id="deadband.switch" compact />
    <label className="row" style={{ fontWeight: 500, color: 'var(--text)' }}>
      <input type="checkbox" checked={deadbands.enabled}
        onChange={event => perform(() => api(`/ieds/${ied.id}/deadbands`, 'PUT', { enabled: event.target.checked }))} />
      <span>Use SCD deadbands</span></label>
    <p className="small muted" style={{ margin: 0 }}>{deadbands.enabled
      ? `On: ${countOf(deadbands.applied, 'measurement')} report only bigger changes.`
      : `Off: every change of the ${countOf(count, 'measurement')} with a db is reported.`}
      {deadbands.without_range ? ` ${countOf(deadbands.without_range, 'measurement')} ha${deadbands.without_range === 1 ? 's' : 've'} no measuring range (rangeC), so never a deadband.` : ''}</p>
    <Disclosure title={`Deadband of each measurement · ${count}`}>
      <div className="table-scroll" style={{ maxHeight: 320 }}><table className="compact-rows">
        <thead><tr><th>Measurement</th><th>db</th><th>What it does when on</th><th>SCADA may change db</th></tr></thead>
        <tbody>{deadbands.points.map(point => <tr key={point.ref}>
          <td><code>{point.object}</code></td><td>{point.db}</td>
          <td className="small">{point.note}</td><td className="small">{point.db_writable ? 'Yes' : 'No (read-only in the SCD)'}</td>
        </tr>)}</tbody></table></div>
    </Disclosure>
  </div>;
}
