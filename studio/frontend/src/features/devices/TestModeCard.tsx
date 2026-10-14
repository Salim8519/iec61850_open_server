import { api } from '../../api/client';
import type { Ied } from '../../api/types';
import { MODE_CHOICES, modeStatus } from '../../copy/testMode';
import { Hint } from '../../ui/Hint';
import { StatusPill } from '../../ui/StatusPill';
import type { ModeSummary } from './useModes';

type Settings = { devices: Record<string, number>; simulation: boolean };

/** Simulated IEDs: choose each logical device's mode and simulation. Real IEDs: read-only badges. */
export function TestModeCard({ ied, summary, perform }: { ied: Ied; summary?: ModeSummary;
  perform: (task: () => Promise<unknown>) => Promise<void> }) {
  const devices = Object.keys(summary?.devices ?? {});
  if (ied.role === 'real') return summary ? <div className="guide-section"><h3 style={{ margin: 0 }}>Mode (read from the device)</h3>
    <Hint id="mode.real" compact />
    <div className="row" style={{ flexWrap: 'wrap' }}>{devices.map(domain => <span key={domain} className="row">
      <code>{domain}</code><StatusPill status={modeStatus(summary.devices[domain])} /></span>)}
      {summary.simulation && <StatusPill status={modeStatus('simulation')} showHint />}</div></div> : null;
  if (ied.role !== 'simulated' || !devices.length) return null;
  const saved: Settings = { devices: {}, simulation: false, ...(ied as Ied & { test_mode?: Settings }).test_mode };
  const save = (next: Settings) => perform(() => api(`/ieds/${ied.id}/test-mode`, 'PUT', next));
  return <div className="guide-section"><h3 style={{ margin: 0 }}>Test mode of this simulated IED</h3>
    <Hint id="mode.simulated" compact />
    <div className="form-grid">{devices.map(domain => <label key={domain}><code>{domain}</code>
      <select value={saved.devices[domain] ?? 1} onChange={event => save({ ...saved,
        devices: { ...saved.devices, [domain]: Number(event.target.value) } })}>
        {MODE_CHOICES.map(choice => <option key={choice.value} value={choice.value}>{choice.label}</option>)}</select></label>)}</div>
    <label className="row" style={{ fontWeight: 500, color: 'var(--text)' }}>
      <input type="checkbox" checked={saved.simulation} onChange={event => save({ ...saved, simulation: event.target.checked })} />
      <span>Simulation mode (LPHD.Sim): listen to test equipment instead of the real messages</span></label>
  </div>;
}
