import type { Ied, Project } from '../api/types';

/** Whether nothing, some or every included IED runs (change 0106). Left-out IEDs never count,
 *  like the top bar words (ADR-0067). The same rule feeds the top bar, sidebar, Devices and SLD. */
export type RunShape = 'stopped' | 'partial' | 'all';
export type RunSummary = { shape: RunShape; running: Ied[]; stopped: Ied[]; total: number };

/** A shape as well as a colour, so the state never depends on colour alone. */
export const RUN_MARK: Record<RunShape, string> = { stopped: '○', partial: '◐', all: '●' };

export function runSummary(project: Pick<Project, 'ieds' | 'runtime' | 'running_ieds'> | null): RunSummary {
  const included = (project?.ieds ?? []).filter(ied => ied.role !== 'off');
  const on = project?.runtime === 'running' || project?.runtime === 'paused';
  const ids = new Set(on ? project?.running_ieds ?? included.map(ied => ied.id) : []);
  const running = included.filter(ied => ids.has(ied.id));
  const stopped = included.filter(ied => !ids.has(ied.id));
  const shape: RunShape = !running.length ? 'stopped' : stopped.length ? 'partial' : 'all';
  return { shape, running, stopped, total: included.length };
}

/** "C03_Y1105" · "A, B" · "A, B, C and 2 more": full names, never cut. */
export function nameList(ieds: Ied[], max = 3): string {
  const names = ieds.map(ied => ied.name);
  return names.length <= max ? names.join(', ') : `${names.slice(0, max).join(', ')} and ${names.length - max} more`;
}

export const RUN_SHAPE_COPY = {
  strip: (summary: RunSummary) =>
    `Live: ${summary.running.length} of ${summary.total} IEDs · ${nameList(summary.running)} · drawing locked while running`,
  devices: (summary: RunSummary) =>
    `${summary.running.length} of ${summary.total} running: ${nameList(summary.running)}. ${summary.stopped.length === 1
      ? 'The other one is stopped.' : `The other ${summary.stopped.length} are stopped.`}`,
  runRest: (count: number) => count === 1 ? 'Run the other one' : `Run the other ${count}`,
  equipmentStopped: 'IED stopped',
  equipmentStoppedHint: (names: string) =>
    `${names} is stopped, so this equipment has no live value. Run it from Devices, or press Run all.`,
  diagramPartial: (running: number, total: number) => `Live diagram: ${running} of ${total} IEDs running`,
  iedsPartial: (running: number, total: number) => `${running} of ${total} IEDs running`,
};
