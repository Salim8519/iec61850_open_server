import { Play } from 'lucide-react';
import { api } from '../../api/client';
import type { Project } from '../../api/types';
import { RUN_MARK, RUN_SHAPE_COPY, runSummary } from '../../copy/runShape';

/** Devices while only some IEDs run (change 0106): one line saying who is live, and the way
 *  to start the rest. Nothing shows when everything is stopped or everything runs. */
export function DeviceRunSummary({ project, perform }: {
  project: Project; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const summary = runSummary(project);
  if (summary.shape !== 'partial') return null;
  return <div className="device-run-summary" role="status">
    <span className="run-mark" aria-hidden>{RUN_MARK.partial}</span>
    <span className="device-run-words">{RUN_SHAPE_COPY.devices(summary)}</span>
    <button onClick={() => perform(() => api('/runtime/start', 'POST'))}
      title="Start every stopped IED that is not skipped; running IEDs keep going">
      <Play size={15} />{RUN_SHAPE_COPY.runRest(summary.stopped.length)}</button>
  </div>;
}
