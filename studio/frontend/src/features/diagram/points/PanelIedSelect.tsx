import type { EquipmentData, Ied } from '../../../api/types';
import { POINT_COPY } from '../../../copy/pointPanel';

/** The one IED of a point panel. It can change only while the panel has no points: a panel never mixes IEDs. */
export function PanelIedSelect({ data, ieds, locked, update }: {
  data: EquipmentData; ieds: Ied[]; locked: boolean; update: (data: EquipmentData) => void;
}) {
  const held = (data.points?.length ?? 0) > 0;
  return <label className="pp-ied-select">IED
    <select value={data.ied_id ?? ''} disabled={locked || held} onChange={event => update({ ...data, ied_id: event.target.value || null })}>
      <option value="">{POINT_COPY.chooseIed}</option>
      {ieds.map(ied => <option key={ied.id} value={ied.id}>{ied.name}</option>)}
    </select>
    <small className="muted">{held ? POINT_COPY.iedLocked : POINT_COPY.iedHint}</small>
  </label>;
}
