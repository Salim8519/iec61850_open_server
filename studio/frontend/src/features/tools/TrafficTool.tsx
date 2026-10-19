import type { Project } from '../../api/types';
import { GooseTraffic } from '../network/GooseTraffic';
import { SvTraffic } from '../network/SvTraffic';

/** GOOSE (inside ScadaBench and from the network) and Sampled Values (lab adapter) of the project. */
export function TrafficTool({ project, locked, perform }: {
  project: Project; locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const running = project.runtime !== 'stopped';
  const lab = project.network.mode === 'lab';
  return <>
    <GooseTraffic running={running} revision={project.revision} />
    <SvTraffic running={running} revision={project.revision} lab={lab} locked={locked} perform={perform} />
  </>;
}
