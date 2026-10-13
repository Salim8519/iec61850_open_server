import { useRef, useState } from 'react';
import { X } from 'lucide-react';
import { useStudio } from '../state/useStudio';
import { ImportSelection } from '../features/ieds/ImportSelection';
import { EventLog } from '../features/event-log/EventLog';
import { RecoveryNotice } from '../components/RecoveryNotice';
import { MigrationNotice } from '../components/MigrationNotice';
import { DiagramPage } from '../features/diagram/DiagramPage';
import { DevicesPage } from '../features/devices/DevicesPage';
import { TestPage } from '../features/test/TestPage';
import { SettingsPage } from '../features/settings/SettingsPage';
import { Sidebar } from './Sidebar';
import { TopBar } from './TopBar';
import { place, type PlaceId } from './places';

export function App() {
  const studio = useStudio();
  const [current, setCurrent] = useState<PlaceId>('diagram');
  const [activity, setActivity] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);
  const project = studio.project;
  const locked = project?.runtime !== 'stopped';
  const openFile = () => fileInput.current?.click();
  const info = place(current);
  return <div className="app">
    {studio.importPreview && <ImportSelection preview={studio.importPreview} busy={studio.busy}
      error={studio.error} confirm={studio.confirmImport} cancel={studio.cancelImport} />}
    <Sidebar current={current} go={setCurrent} connected={!!project} eventCount={studio.events.length}
      openActivity={() => setActivity(true)} />
    <div className="main">
      <TopBar title={info.title} subtitle={info.subtitle} project={project} busy={studio.busy}
        saveState={studio.saveState} run={studio.runtime} openFile={openFile} save={studio.save} />
      <RecoveryNotice recovery={project?.startup_recovery} />
      <MigrationNotice notice={project?.migration_notice} />
      {studio.error && <div className="error-banner" role="alert"><span>{studio.error}</span>
        <button className="ghost" onClick={() => studio.setError('')} aria-label="Dismiss error"><X size={16} /></button></div>}
      <input ref={fileInput} type="file" accept=".cid,.icd,.scd,.iid,.subsim" style={{ display: 'none' }}
        aria-label="Open CID, SCD or project file" onChange={event => {
          const file = event.target.files?.[0]; if (file) studio.importFile(file); event.target.value = '';
        }} />
      <main className="page">
        {project && current === 'diagram' && <DiagramPage studio={studio} project={project} locked={locked}
          openDevices={() => setCurrent('devices')} />}
        {project && current === 'devices' && <DevicesPage project={project} locked={locked} values={studio.values}
          perform={studio.perform} prepare={studio.prepare} importFile={openFile} addTemplate={studio.addTemplate} />}
        {project && current === 'test' && <TestPage project={project} locked={locked} values={studio.values}
          events={studio.events} perform={studio.perform} prepare={studio.prepare} />}
        {project && current === 'settings' && <SettingsPage project={project} locked={locked} perform={studio.perform} />}
      </main>
    </div>
    {activity && <EventLog events={studio.events} close={() => setActivity(false)}
      session={project ? `${project.session_id.slice(0, 8)} · revision ${project.revision}` : undefined} />}
  </div>;
}
