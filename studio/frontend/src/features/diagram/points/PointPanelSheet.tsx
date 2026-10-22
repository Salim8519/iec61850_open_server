import { useContext } from 'react';
import { ListChecks } from 'lucide-react';
import type { EquipmentData, Project } from '../../../api/types';
import { OpenPointPanel } from '../../../equipment/pointPanelContext';
import { POINT_COPY } from '../../../copy/pointPanel';
import { PanelIedSelect } from './PanelIedSelect';

/** Properties of a point panel: its IED, how many points it holds, and the button that opens the popup. */
export function PointPanelSheet({ nodeId, data, project, locked, update }: {
  nodeId: string; data: EquipmentData; project: Project; locked: boolean; update: (data: EquipmentData) => void;
}) {
  const open = useContext(OpenPointPanel);
  const count = data.points?.length ?? 0;
  return <div className="relay-editor">
    <PanelIedSelect data={data} ieds={project.ieds} locked={locked} update={update} />
    <p className="small">{count} point{count === 1 ? '' : 's'} in this panel.</p>
    <button className="primary" disabled={!data.ied_id} onClick={() => open(nodeId)}><ListChecks size={16} />
      {locked ? 'Open points' : count ? 'Open points' : POINT_COPY.addPoints}</button>
    {!data.ied_id && <p className="small muted">{POINT_COPY.chooseIed}.</p>}
  </div>;
}
