import { useCallback, useEffect, useRef, useState } from 'react';
import { X } from 'lucide-react';
import { useStudio } from '../state/useStudio';
import { OPEN_FILE_ACCEPT } from '../state/projectFile';
import { ImportSelection } from '../features/ieds/ImportSelection';
import { EventLog } from '../features/event-log/EventLog';
import { RecoveryNotice } from '../components/RecoveryNotice';
import { MigrationNotice } from '../components/MigrationNotice';
import { DiagramPage } from '../features/diagram/DiagramPage';
import { DevicesPage } from '../features/devices/DevicesPage';
import { TestPage } from '../features/test/TestPage';
import { SettingsPage } from '../features/settings/SettingsPage';
import { ToolsPage } from '../features/tools/ToolsPage';
import type { ToolId } from '../copy/tools';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';
import { WorkPanel } from './WorkPanel';
import { TourOverlay } from '../features/tour/TourOverlay';
import { getPreferences, markTourSeen } from '../api/preferences';
import { place, type PlaceId } from './places';
import { useExitConfirm } from './useExitConfirm';
import { placeActivity, toolActivity } from './running';
import { useActivity } from './useActivity';
import { useUpdateStatus } from '../api/updates';
import { GooseConflicts } from '../features/network/GooseConflicts';

export function App() {
  const studio = useStudio();
  useExitConfirm();
  const [current, setCurrent] = useState<PlaceId>('devices');
  const [activity, setActivity] = useState(false);
  const [flowIed, setFlowIed] = useState<string | undefined>();
  const [tool, setTool] = useState<ToolId | null>(null);
  const [tour, setTour] = useState<number | null>(null);
  const [tourVersion, setTourVersion] = useState(1);
  useEffect(() => {
    getPreferences().then(prefs => {
      setTourVersion(prefs.tour_version);
      if (prefs.tour_seen_version < prefs.tour_version) setTour(prefs.tour_version);
    })
      .catch(() => {});
  }, []);
  const closeTour = useCallback(() => {
    if (tour !== null) markTourSeen(tour).catch(() => {});
    setTour(null);
  }, [tour]);
  const tourGo = useCallback((next: PlaceId) => { setFlowIed(undefined); setCurrent(next); }, []);
  const fileInput = useRef<HTMLInputElement>(null);
  const project = studio.project;
  const locked = project?.runtime !== 'stopped';
  // Stable callbacks keep the memoised diagram toolbar from being rebuilt on every app render.
  const openFile = useCallback(() => fileInput.current?.click(), []);
  const openDevices = useCallback((iedId?: string) => { setFlowIed(iedId); setCurrent('devices'); }, []);
  const info = place(current);
  const running = useActivity(!!project);
  const update = useUpdateStatus();
  return <div className="app">
    <WorkPanel work={studio.importPreview ? null : studio.work} />
    {studio.importPreview && <ImportSelection preview={studio.importPreview} busy={studio.busy}
      error={studio.error} confirm={studio.confirmImport} cancel={studio.cancelImport} work={studio.work} />}
    <Sidebar current={current} go={next => { setFlowIed(undefined); setCurrent(next); }} connected={!!project}
      openActivity={() => setActivity(true)} openTour={() => setTour(tourVersion)} live={placeActivity(project, running)}
      update={update.status} dismissUpdate={version => { update.change({ dismissed: version }).catch(() => {}); }} />
    <div className="main">
      <TopBar title={info.heading} subtitle={info.subtitle} project={project} busy={studio.busy} working={studio.working}
        saveState={studio.saveState} run={studio.runtime} openFile={openFile} save={studio.save}
        openNetwork={() => setCurrent('settings')} />
      <RecoveryNotice recovery={project?.startup_recovery} />
      <MigrationNotice notice={project?.migration_notice} />
      {studio.error && <div className="error-banner" role="alert"><span>{studio.error}</span>
        <button className="ghost" onClick={() => studio.setError('')} aria-label="Dismiss error"><X size={16} /></button></div>}
      <input ref={fileInput} type="file" accept={OPEN_FILE_ACCEPT} style={{ display: 'none' }}
        aria-label="Open CID, SCD or project file" onChange={event => {
          const file = event.target.files?.[0]; if (file) studio.importFile(file); event.target.value = '';
        }} />
      <main className="page">
        {!project && !studio.error && <div className="empty" role="status">
          Opening your last project… Large stations can take a few seconds.</div>}
        {project && current === 'diagram' && <DiagramPage studio={studio} project={project} locked={locked}
          openDevices={openDevices} openFile={openFile} />}
        {project && current === 'devices' && <DevicesPage project={project} locked={locked}
          perform={studio.perform} prepare={studio.prepare} importFile={openFile} addTemplate={studio.addTemplate}
          flowIed={flowIed} openNetwork={() => setCurrent('settings')} />}
        {project && current === 'test' && <TestPage project={project} locked={locked}
          perform={studio.perform} prepare={studio.prepare} refresh={studio.refresh} />}
        {project && current === 'tools' && <ToolsPage project={project} locked={locked} perform={studio.perform}
          tool={tool} openTool={setTool} running={toolActivity(running)} activity={running} runIeds={() => studio.runtime('start')} />}
        {project && current === 'settings' && <SettingsPage project={project} locked={locked} perform={studio.perform}
          openTraffic={() => { setTool('traffic'); setCurrent('tools'); }} update={update} />}
      </main>
    </div>
    {project && <GooseConflicts project={project} />}
    {tour !== null && <TourOverlay go={tourGo} close={closeTour} />}
    {activity && <EventLog close={() => setActivity(false)}
      session={project ? `${project.session_id.slice(0, 8)} · revision ${project.revision}` : undefined} />}
  </div>;
}
