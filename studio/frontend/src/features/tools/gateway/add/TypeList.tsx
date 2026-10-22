import type { TypeEntry } from '../../../../api/gatewayAdd';
import { ADD_COPY } from '../../../../copy/gatewayAdd';
import { kindOfGatewayPoint, type Kind } from '../../../../points/pointKind';
import { KindBadge } from '../../../../ui/KindBadge';
import { splitTypes } from './addPlan';

type Choice<K extends string> = Pick<TypeEntry, 'name' | 'type_id' | 'most_used'> & { kind: K };

/** The one type list of every protocol (ADR-0105): "Most used" on top, "More types" below, each type with
 *  its badge, its plain name and the protocol's own type ids as small secondary text.
 *  The entries come from the engine's catalog, so a new type shows here without a frontend change.
 *  The gateway passes its kinds; the SCADA master (ADR-0110) passes its own types with `badge`. */
export function TypeList<K extends string>({ types, value, change, label = ADD_COPY.chooseType, badge = kindOfGatewayPoint }: {
  types: Choice<K>[]; value: K | null; change: (kind: K) => void; label?: string; badge?: (kind: K) => Kind;
}) {
  const { most, more } = splitTypes(types);
  const group = (title: string, entries: Choice<K>[]) => entries.length > 0 && <div className="type-group">
    <span className="small muted type-group-title">{title}</span>
    <div className="type-choices">{entries.map(entry => <button key={entry.kind} type="button" role="radio"
      aria-checked={value === entry.kind} className={`type-choice${value === entry.kind ? ' active' : ''}`}
      onClick={() => change(entry.kind)}>
      <KindBadge kind={badge(entry.kind)} raw={entry.type_id} />
      <span className="type-choice-text"><span>{entry.name}</span><small>{entry.type_id}</small></span>
      {value === entry.kind && <span className="sr-only">(chosen)</span>}
    </button>)}</div>
  </div>;
  return <div className="type-list" role="radiogroup" aria-label={label}>
    {group(ADD_COPY.mostUsed, most)}
    {group(ADD_COPY.moreTypes, more)}
  </div>;
}
