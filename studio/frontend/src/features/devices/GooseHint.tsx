import type { GooseLinkMap } from '../../api/signalLayers';
import { gooseHint } from '../../copy/signalLayers';
import { shortNames } from '../../equipment/signalLayout';

/** One quiet line in a Devices list row: "↑ sends to 2 · ↓ gets from C09 · ⚠ 1 not heard". */
export function GooseHint({ map, name }: { map: GooseLinkMap; name: string }) {
  const text = gooseHint(map.summary[name], shortNames(map.ieds.map(ied => ied.name)));
  return text ? <small className="goose-hint" title="GOOSE declared in the SCD; while running, what is really heard">{text}</small> : null;
}
