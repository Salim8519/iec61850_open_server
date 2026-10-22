import { useContext, type KeyboardEvent } from 'react';
import { TriangleAlert } from 'lucide-react';
import type { EquipmentData, PanelPoint } from '../api/types';
import { EquipmentSymbol } from './symbols/EquipmentSymbol';
import { relayValueKey } from './stateBindings';
import { CanvasLocked, OpenPointPanel, useLiveValues } from './pointPanelContext';
import { StoppedIeds } from './liveContext';
import { summarize } from './pointDisplay';
import { hasLiveValue } from './pointPanel';
import { POINT_COPY } from '../copy/pointPanel';
import './point-panel.css';

const SHOWN_PINNED = 5;
const TYPE_WORDS: [keyof ReturnType<typeof summarize>['counts'], string, string][] = [
  ['status', 'status', 'status'], ['double', 'double', 'double'], ['command', 'command', 'commands'], ['value', 'value', 'values']];

/** Canvas projection of a point panel: name, IED, how many points of each type, the pinned values and
 *  a "N not healthy" badge. Clicking it (or Enter) opens the popup with every point. Fixed width;
 *  long text wraps instead of being cut. */
export function PointPanelCard({ nodeId, data, iedNames }: {
  nodeId: string; data: EquipmentData; iedNames: Record<string, string>;
}) {
  const points = data.points || [];
  const open = useContext(OpenPointPanel);
  const locked = useContext(CanvasLocked);
  const stoppedIeds = useContext(StoppedIeds);
  const stopped = !!data.ied_id && stoppedIeds.has(data.ied_id);
  const live = points.filter(point => hasLiveValue(point) && (point.pinned || point.healthy !== null));
  const values = useLiveValues(live.map(point => relayValueKey(nodeId, point.id)));
  const byId = new Map<string, unknown>(live.map((point, index) => [point.id, values[index]]));
  const running = locked && !stopped;
  const summary = summarize(points, (point: PanelPoint) => byId.get(point.id), running);
  const iedName = data.ied_id ? iedNames[data.ied_id] ?? data.ied_id : POINT_COPY.noIed;
  const press = (event: KeyboardEvent) => {
    if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); open(nodeId); }
  };
  return <div className="point-card" role="button" tabIndex={0} title={POINT_COPY.openHint}
    aria-label={`${data.label}, ${points.length} points. ${POINT_COPY.openHint}`}
    onClick={() => open(nodeId)} onKeyDown={press}>
    <div className="point-card-head">
      <span className="point-card-icon"><EquipmentSymbol kind="POINT_PANEL" size={22} /></span>
      <span className="point-card-title"><strong>{data.label}</strong><small>{iedName}</small></span>
    </div>
    {points.length ? <div className="point-card-counts">
      {TYPE_WORDS.filter(([key]) => summary.counts[key]).map(([key, one, many]) => <span key={key} className="point-card-count">
        <b>{summary.counts[key]}</b> {summary.counts[key] === 1 ? one : many}</span>)}
    </div> : <p className="point-card-empty">{POINT_COPY.empty}</p>}
    {summary.pinned.length > 0 && <ul className="point-card-pinned">
      {summary.pinned.slice(0, SHOWN_PINNED).map(item => <li key={item.id} title={`${item.label}: ${item.text}`}>
        <span className="point-card-name">{item.label}</span>
        <span className={`point-chip ${item.state}`}>{item.text}</span></li>)}
      {summary.pinned.length > SHOWN_PINNED && <li className="point-card-more">+{summary.pinned.length - SHOWN_PINNED} more pinned</li>}
    </ul>}
    {summary.notHealthy > 0 && <p className="point-card-unhealthy" role="status">
      <TriangleAlert size={13} aria-hidden="true" /><span>{POINT_COPY.notHealthy(summary.notHealthy)}</span></p>}
  </div>;
}
