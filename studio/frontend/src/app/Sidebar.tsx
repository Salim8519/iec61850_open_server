import { Activity, Cpu, FlaskConical, Settings, Workflow, Zap } from 'lucide-react';
import { PLACES, type PlaceId } from './places';

const ICONS: Record<PlaceId, typeof Workflow> = { diagram: Workflow, devices: Cpu, test: FlaskConical, settings: Settings };

export function Sidebar({ current, go, connected, eventCount, openActivity }: {
  current: PlaceId; go: (id: PlaceId) => void; connected: boolean; eventCount: number; openActivity: () => void;
}) {
  const item = (id: PlaceId) => {
    const Icon = ICONS[id];
    const entry = PLACES.find(placeItem => placeItem.id === id)!;
    return <button key={id} className={`nav-item${current === id ? ' active' : ''}`} aria-current={current === id ? 'page' : undefined}
      title={entry.title} onClick={() => go(id)}><Icon size={21} /><span>{entry.title}</span></button>;
  };
  return <nav className="sidebar" aria-label="Main">
    <div className="brand"><span className="brand-mark"><Zap size={19} /></span>
      <span>Substation Studio<small>IEC 61850 test lab</small></span></div>
    {PLACES.filter(entry => entry.id !== 'settings').map(entry => item(entry.id))}
    <div className="sidebar-spacer" />
    <div className="sidebar-foot">
      {item('settings')}
      <button className="nav-item" onClick={openActivity} title="Activity">
        <Activity size={19} /><span>Activity ({eventCount})</span></button>
      <span className="small muted" style={{ padding: '4px 12px' }} title={connected ? 'Engine connected' : 'Connecting'}>
        <span className="dot" style={{ display: 'inline-block', marginRight: 8, color: connected ? 'var(--success)' : 'var(--warning)' }} />
        <span>{connected ? 'Engine connected' : 'Connecting…'}</span></span>
    </div>
  </nav>;
}
