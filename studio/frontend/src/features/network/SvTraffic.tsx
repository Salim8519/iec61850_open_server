import { Activity } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import { useLiveFeed } from '../../api/useLiveFeed';
import { svStatus } from '../../copy/sv';
import { UI_STATUS } from '../../copy/status';
import { EmptyState } from '../../ui/EmptyState';
import { Hint } from '../../ui/Hint';
import { SegmentedControl } from '../../ui/SegmentedControl';
import { StatusPill } from '../../ui/StatusPill';

type Channel = { name: string; rms: number; magnitude: number; angle: number };
type Stream = { key: string; publisher: string; sv_id: string; app_id: number; mac: string; smp_rate: number;
  smp_mod: number; expected_rate: number | null; layout: string; state: string; problems: string[];
  received_rate?: number; received_lost?: number; received_smp_synch?: number; received_values?: Channel[];
  received_raw?: number[]; observed_at?: string };
type Overview = { running: boolean; state: string; message: string; nominal_hz: number; watches: Stream[];
  unknown: { app_id: number; mac: string; sv_id: string; simulate: boolean }[]; not_supervised: { key: string; publisher: string }[] };

const SYNC = ['not synchronised', 'local clock', 'global clock'];
const unit = (name: string) => name.includes('TVTR') || name.includes('Vol') ? 'V' : 'A';

/** Sampled Values streams from the SCD and what is really on the lab adapter (listen only). */
export function SvTraffic({ running, revision, lab, locked, perform }: { running: boolean; revision: number; lab: boolean;
  locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void> }) {
  const overview = useLiveFeed<Overview>('/sv', running ? 1000 : null, revision).value;
  const [open, setOpen] = useState('');
  if (!overview) return null;
  const selected = overview.watches.find(item => item.key === open);
  return <div className="card">
    <div className="card-heading"><h2>Sampled Values</h2>
      <SegmentedControl label="Nominal frequency" value={String(overview.nominal_hz)} disabled={locked}
        options={[{ value: '50', label: '50 Hz' }, { value: '60', label: '60 Hz' }]}
        onChange={value => perform(() => api('/network', 'PUT', { nominal_hz: Number(value) }))} /></div>
    <div className="guide-section"><Hint id={lab ? 'sv.intro' : 'sv.needs_lab'} /><Hint id="sv.frequency" compact />
      {overview.message && running && <p className="small muted" style={{ margin: 0 }}>{overview.message}</p>}</div>
    {!overview.watches.length ? <EmptyState icon={<Activity size={20} />} title="No Sampled Values in this project" hintId="sv.intro" />
      : <table><thead><tr><th>Stream</th><th>From</th><th>APPID / MAC</th><th>Status</th><th>Rate</th><th>Lost</th><th>Clock</th></tr></thead>
        <tbody>{overview.watches.map(stream => <tr key={stream.key} className={open === stream.key ? 'active' : ''}
          onClick={() => setOpen(open === stream.key ? '' : stream.key)} style={{ cursor: 'pointer' }}>
          <td><strong>{stream.sv_id || stream.key}</strong><div className="small muted"><code>{stream.key}</code></div></td>
          <td>{stream.publisher}</td>
          <td><code>{`0x${stream.app_id.toString(16).toUpperCase().padStart(4, '0')}`}</code><div className="small"><code>{stream.mac}</code></div></td>
          <td>{lab ? <StatusPill status={svStatus(stream.state, stream.problems)} showHint={stream.state !== 'receiving'} />
            : <StatusPill status={UI_STATUS.sv_needs_lab} />}</td>
          <td className="small">{stream.received_rate ? `${Math.round(stream.received_rate)}/s` : '–'}
            <div className="muted">SCD {stream.expected_rate ? `${stream.expected_rate}/s` : 'not stated'}</div></td>
          <td className="small">{stream.received_lost ?? '–'}</td>
          <td className="small">{stream.received_smp_synch === undefined || stream.received_smp_synch === null ? '–'
            : SYNC[stream.received_smp_synch] ?? `source ${stream.received_smp_synch}`}</td></tr>)}</tbody></table>}
    {selected && <div className="guide-section">
      <h3 style={{ margin: 0 }}>{selected.sv_id} · measured values (nominal {overview.nominal_hz} Hz)</h3>
      {selected.layout !== '9-2LE' ? <p className="small">Not interpreted: this dataset is not a 9-2LE layout, so only raw numbers are shown.
        {selected.received_raw?.length ? ` First values: ${selected.received_raw.slice(0, 8).join(', ')}` : ''}</p>
        : <table><thead><tr><th>Channel</th><th>RMS</th><th>Fundamental</th><th>Angle</th></tr></thead><tbody>
          {(selected.received_values ?? []).map(channel => <tr key={channel.name}><td><code>{channel.name}</code></td>
            <td>{channel.rms.toFixed(3)} {unit(channel.name)}</td><td>{channel.magnitude.toFixed(3)} {unit(channel.name)}</td>
            <td>{channel.angle.toFixed(1)}°</td></tr>)}</tbody></table>}
      {!selected.received_values?.length && selected.layout === '9-2LE' && <p className="small muted">Values appear after one full cycle is received.</p>}
    </div>}
    {!!overview.not_supervised.length && <div className="guide-section"><Hint id="sv.limit" />
      <ul className="small">{overview.not_supervised.map(item => <li key={item.key}><code>{item.key}</code> ({item.publisher})</li>)}</ul></div>}
    {!!overview.unknown.length && <div className="guide-section"><Hint id="sv.unknown" />
      <ul className="small">{overview.unknown.map(item => <li key={`${item.app_id}-${item.mac}`}>{item.sv_id || '(no svID)'} · APPID {item.app_id} · {item.mac}
        {item.simulate ? ' · simulation' : ''}</li>)}</ul></div>}
  </div>;
}
