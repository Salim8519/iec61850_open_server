import { Activity, CircleHelp, Compass, Cpu, FlaskConical, PlayCircle, Settings, Workflow, Wrench } from 'lucide-react';
import { Menu } from '../ui/Menu';
import { TOUR_COPY, TUTORIALS_URL } from '../copy/tour';
import { PLACES, type PlaceId } from './places';
import type { Running } from './running';
import type { UpdateStatus } from '../api/updates';
import { UpdateNote } from './UpdateNote';
import { useEventCount } from '../state/useLive';

/** Only this number re-renders when events arrive, not the sidebar or the app. */
function ActivityCount() {
  return <>{useEventCount()}</>;
}

const ICONS: Record<PlaceId, typeof Workflow> = { diagram: Workflow, devices: Cpu, test: FlaskConical, tools: Wrench,
  settings: Settings };

export function Sidebar({ current, go, connected, openActivity, openTour, live, update, dismissUpdate }: {
  current: PlaceId; go: (id: PlaceId) => void; connected: boolean; openActivity: () => void;
  openTour: () => void; live: Partial<Record<PlaceId, Running>>;
  update?: UpdateStatus | null; dismissUpdate?: (version: string) => void;
}) {
  const item = (id: PlaceId) => {
    const Icon = ICONS[id];
    const entry = PLACES.find(placeItem => placeItem.id === id)!;
    const running = live[id];
    const words = running ? running.labels.join(' · ') : '';
    return <button key={id} className={`nav-item${current === id ? ' active' : ''}`} aria-current={current === id ? 'page' : undefined}
      title={running ? `${entry.heading}: ${words}` : entry.heading} data-tour={`place-${id}`} onClick={() => go(id)}>
      <Icon size={21} /><span>{entry.title}</span>
      {running && <><span className={`live-dot${running.paused ? ' paused' : ''}${running.partial ? ' partial' : ''}`} data-live={id} aria-hidden />
        <span className="sr-only">: {words}</span></>}</button>;
  };
  return <nav className="sidebar" aria-label="Main">
    <div className="brand"><img className="brand-mark" src="/assets/brand/favicon.svg" alt="" width={34} height={34} />
      <span>ScadaBench<small>IEC 61850 test bench</small></span></div>
    {PLACES.filter(entry => entry.id !== 'settings').map(entry => item(entry.id))}
    <div className="sidebar-spacer" />
    <div className="sidebar-foot">
      {dismissUpdate && <UpdateNote status={update ?? null} dismiss={dismissUpdate} />}
      {item('settings')}
      <Menu label={TOUR_COPY.helpTitle} align="left" up triggerClass="nav-item" triggerTour="help"
        trigger={<><CircleHelp size={19} /><span>{TOUR_COPY.helpTitle}</span></>} items={[
          { label: TOUR_COPY.takeTour, icon: <Compass size={16} />, onSelect: openTour },
          { label: TOUR_COPY.tutorials, icon: <PlayCircle size={16} />,
            onSelect: () => { window.open(TUTORIALS_URL, '_blank', 'noopener'); } },
        ]} />
      <button className="nav-item" onClick={openActivity} title="Activity">
        <Activity size={19} /><span>Activity (<ActivityCount />)</span></button>
      <span className="small muted engine-status" style={{ padding: '4px 12px' }} title={connected ? 'Engine connected' : 'Connecting'}>
        <span className="dot" style={{ display: 'inline-block', marginRight: 8, color: connected ? 'var(--success)' : 'var(--warning)' }} />
        <span>{connected ? 'Engine connected' : 'Connecting…'}</span></span>
    </div>
  </nav>;
}
