import { KindBadge } from '../ui/KindBadge';
import type { Kind } from '../points/pointKind';
import './type-cell.css';

/** FC tag: the small grey functional constraint that stays beside the type badge (ADR-0095). */
export function FcTag({ fc }: { fc: string }) {
  return <span className="badge fc-tag" title={`Functional constraint ${fc}: the group of data this point belongs to`}>{fc}</span>;
}

/** The type column of a point table: the kind badge (hover for the name and raw type), then the FC tag. */
export function TypeCell({ kind, raw, fc }: { kind: Kind; raw?: string; fc?: string }) {
  return <td className="type-cell"><KindBadge kind={kind} raw={raw} />{fc && <FcTag fc={fc} />}</td>;
}
