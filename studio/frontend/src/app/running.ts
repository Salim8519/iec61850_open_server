import type { Activity } from '../api/activity';
import type { Project } from '../api/types';
import type { ToolId } from '../copy/tools';
import { RUNNING_COPY } from '../copy/running.ts';
import type { PlaceId } from './places';

export type Running = { labels: string[]; paused?: boolean };

/** What runs in each tool, in words, for the tool cards. Only running tools are listed. */
export function toolActivity(activity: Activity | null): Partial<Record<ToolId, string>> {
  if (!activity) return {};
  const result: Partial<Record<ToolId, string>> = {};
  const { gateway, explorer, scan } = activity;
  const { master } = activity;
  const labels = [gateway.state === 'running' && RUNNING_COPY.gateway(gateway.protocol, gateway.port, gateway.clients),
    master && (master.state === 'connected' || master.state === 'lost')
      && RUNNING_COPY.master(master.protocol, master.host, master.points, master.state === 'lost')].filter(Boolean);
  if (labels.length) result.gateway = labels.join(' · ');
  if (explorer.state === 'connected') result.explorer = RUNNING_COPY.explorer(explorer.host, explorer.watched, explorer.polling);
  if (scan.state === 'scanning') result.find = RUNNING_COPY.scan(scan.probed, scan.total);
  return result;
}

/** One running dot per place, each with its own meaning: IED Run, sequences, tools. */
export function placeActivity(project: Project | null, activity: Activity | null): Partial<Record<PlaceId, Running>> {
  const result: Partial<Record<PlaceId, Running>> = {};
  const runtime = project?.runtime ?? 'stopped';
  if (project && (runtime === 'running' || runtime === 'paused')) {
    const count = project.ieds.filter(ied => ied.role !== 'off').length;
    const paused = runtime === 'paused';
    result.devices = { labels: [paused ? RUNNING_COPY.iedsPaused(count) : RUNNING_COPY.ieds(count)], paused };
    result.diagram = { labels: [RUNNING_COPY.diagram], paused };
  }
  if (activity?.sequences) result.test = { labels: [RUNNING_COPY.sequences(activity.sequences)] };
  const tools = Object.values(toolActivity(activity));
  if (tools.length) result.tools = { labels: tools };
  return result;
}
