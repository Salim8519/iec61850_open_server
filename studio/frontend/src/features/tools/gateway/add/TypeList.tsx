import type { PointKind } from '../../../../api/gateway';
import type { TypeEntry } from '../../../../api/gatewayAdd';
import { ADD_COPY } from '../../../../copy/gatewayAdd';
import { kindOfGatewayPoint } from '../../../../points/pointKind';
import { KindBadge } from '../../../../ui/KindBadge';
import { splitTypes } from './addPlan';

/** The one type list of every protocol (ADR-0105): "Most used" on top, "More types" below, each type with
 *  its badge, its plain name and the protocol's own type ids as small secondary text.
 *  The entries come from the engine's catalog, so a new type shows here without a frontend change. */
export function TypeList({ types, value, change, label = ADD_COPY.chooseType }: {
  types: TypeEntry[]; value: PointKind | null; change: (kind: PointKind) => void; label?: string;
}) {
  const { most, more } = splitTypes(types);
  const group = (title: string, entries: TypeEntry[]) => entries.length > 0 && <div className="type-group">
    <span className="small muted type-group-title">{title}</span>
    <div className="type-choices">{entries.map(entry => <button key={entry.kind} type="button" role="radio"
      aria-checked={value === entry.kind} className={`type-choice${value === entry.kind ? ' active' : ''}`}
      onClick={() => change(entry.kind)}>
      <KindBadge kind={kindOfGatewayPoint(entry.kind)} raw={entry.type_id} />
      <span className="type-choice-text"><span>{entry.name}</span><small>{entry.type_id}</small></span>
      {value === entry.kind && <span className="sr-only">(chosen)</span>}
    </button>)}</div>
  </div>;
  return <div className="type-list" role="radiogroup" aria-label={label}>
    {group(ADD_COPY.mostUsed, most)}
    {group(ADD_COPY.moreTypes, more)}
  </div>;
}
