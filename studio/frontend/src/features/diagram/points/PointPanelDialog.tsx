import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { CheckCheck, Info, Plus, X } from 'lucide-react';
import type { EquipmentData, EquipmentNode, PanelPoint, Project } from '../../../api/types';
import { StoppedIeds } from '../../../equipment/liveContext';
import { EquipmentSymbol } from '../../../equipment/symbols/EquipmentSymbol';
import { chipCounts, FILTER_CHIPS, filterPoints, listItems, type PointFilter } from '../../../equipment/pointFilter';
import { addPoints, patchPoint, type PointPatch } from '../../../equipment/pointPanel';
import { pointAvailability } from '../../../equipment/pointRequests';
import { FilterBar } from '../../../ui/FilterBar';
import type { FilterSpec, FilterValues } from '../../../ui/filterState';
import { POINT_COPY } from '../../../copy/pointPanel';
import { PanelIedSelect } from './PanelIedSelect';
import { PointAdder } from './PointAdder';
import { PointRow } from './PointRow';
import { usePointActions } from './usePointActions';
import { useTypedControls } from './useTypedControls';
import './point-dialog.css';

const SPEC: FilterSpec = { search: POINT_COPY.searchPlaceholder, groups: [{ key: 'type', label: 'Type of point', initial: 'all',
  options: FILTER_CHIPS.map(chip => ({ value: chip.value, label: chip.label })) }] };
const START: FilterValues = { text: '', type: 'all' };

type Props = {
  node: EquipmentNode; project: Project; locked: boolean; running: ReadonlySet<string>;
  update: (data: EquipmentData) => void; close: () => void;
};

/** The popup of a point panel: filter chips, search, the points grouped and scrollable, and a quick way
 *  to set or operate each one while the IEDs run; rename, group, pin, healthy values and removal while they
 *  are stopped. Opens from the canvas card. */
export function PointPanelDialog({ node, project, locked, running, update, close }: Props) {
  const dialog = useRef<HTMLDialogElement>(null);
  const data = node.data;
  const points = useMemo(() => data.points || [], [data.points]);
  const ied = project.ieds.find(item => item.id === data.ied_id);
  const extended = useTypedControls(ied);
  const available = pointAvailability(ied, running);
  const stopped = !ied || !running.has(ied.id);
  const actions = usePointActions(node.id);
  const [values, setValues] = useState<FilterValues>(START);
  const [adding, setAdding] = useState(false);
  useEffect(() => { dialog.current?.showModal(); }, []);
  const set = useCallback((key: string, value: string | boolean) => setValues(current => ({ ...current, [key]: value })), []);
  const shown = useMemo(() => filterPoints(points, values.type as PointFilter, String(values.text)), [points, values]);
  const items = useMemo(() => listItems(shown), [shown]);
  const setPoints = (next: PanelPoint[]) => update({ ...data, points: next });
  const why = !locked ? POINT_COPY.runToChange : available.reason === 'real' ? POINT_COPY.real
    : available.reason === 'off' ? POINT_COPY.off : available.reason === 'missing' ? POINT_COPY.missing : POINT_COPY.runToChange;
  const withHealthy = points.filter(point => point.healthy !== null && point.type !== 'command').length;
  const note = !locked ? POINT_COPY.editHint : !available.ok ? why : '';
  const stoppedIeds = useMemo(() => new Set(stopped && ied ? [ied.id] : []), [stopped, ied]);
  return <StoppedIeds.Provider value={stoppedIeds}>
    <dialog ref={dialog} className="pp-dialog" aria-labelledby="pp-title" onClose={close}>
      <div className="pp-head">
        <span className="point-card-icon"><EquipmentSymbol kind="POINT_PANEL" size={24} /></span>
        <h2 id="pp-title">{data.label}<small>{ied ? ied.name : POINT_COPY.noIed}</small></h2>
        <span className="pp-spacer" />
        <button disabled={!locked || !available.ok || !withHealthy || actions.working}
          title={locked ? POINT_COPY.healthyHint : POINT_COPY.runToChangeHint} onClick={actions.healthy}>
          <CheckCheck size={16} />{POINT_COPY.allHealthy}{withHealthy ? ` (${withHealthy})` : ''}</button>
        <button className="primary" disabled={locked} aria-pressed={adding}
          title={locked ? 'Stop the IEDs to add points' : 'Pick signals of the IED and add them as points'}
          onClick={() => setAdding(!adding)}><Plus size={16} />{adding ? 'Back to the points' : POINT_COPY.addPoints}</button>
        <button className="icon ghost" aria-label="Close the point panel" onClick={() => dialog.current?.close()}><X size={18} /></button>
      </div>
      {note && <p className="pp-note small"><Info size={14} aria-hidden="true" />{note}</p>}
      {locked && actions.note && <p className="pp-note small" role="status">{actions.note}</p>}
      {adding && !locked ? (ied ? <PointAdder ied={ied} extended={extended} points={points} done={() => setAdding(false)}
        add={fresh => { const result = addPoints(points, fresh); setPoints(result.points); return result.skipped; }} />
        : <div className="pp-choose"><PanelIedSelect data={data} ieds={project.ieds} locked={false} update={update} /></div>) : <>
        <FilterBar label="Filter the points" spec={SPEC} values={values} set={set} clear={() => setValues(START)}
          shown={shown.length} total={points.length} counts={{ type: chipCounts(points) }} />
        <div className="pp-list">
          {!points.length && <div className="pp-empty"><strong>{POINT_COPY.noPoints}</strong>
            <p>{ied || !locked ? POINT_COPY.noPointsHint : POINT_COPY.missing}</p></div>}
          {points.length > 0 && !shown.length && <p className="pp-empty">{POINT_COPY.noMatch}</p>}
          {items.map(item => item.kind === 'group'
            ? <h3 key={item.key} className="pp-group">{item.group || POINT_COPY.noGroup}<small>{item.count}</small></h3>
            : <div key={item.key} className="pp-item">
              <PointRow nodeId={node.id} point={item.point} ied={ied} extended={extended} locked={locked} available={available} why={why}
                stopped={stopped} outcome={actions.outcomes[item.point.id]} set={actions.set} send={actions.send}
                change={(id: string, patch: PointPatch) => setPoints(patchPoint(points, id, patch))}
                remove={id => setPoints(points.filter(point => point.id !== id))} /></div>)}
        </div></>}
    </dialog>
  </StoppedIeds.Provider>;
}
