import { KIND_TEXT } from '../copy/pointKinds';
import { isCommandKind, type Kind } from '../points/pointKind';
import './kind.css';

/** The one point-type badge (ADR-0095): letters + symbol, colour from the shared kind tokens.
 *  `raw` is the protocol's own type (M_DP_TB_1, Gauge32, Dbpos), added to the hover text. */
export function KindBadge({ kind, raw, showName }: { kind: Kind; raw?: string; showName?: boolean }) {
  const text = KIND_TEXT[kind];
  const style = isCommandKind(kind) ? ' command' : kind === 'AT' || kind === 'UN' ? ' quiet' : '';
  return <span className="kind-cell">
    <span className={`kind-badge kind-${kind}${style}`} title={raw ? `${text.name} (${raw})` : text.name}>
      <span aria-hidden>{text.glyph}</span>{kind}
    </span>
    {showName && <span className="kind-name">{text.name}</span>}
  </span>;
}

/** Props for a table row of a known kind: the shared CSS draws its coloured left edge. */
export const kindRow = (kind: Kind) => ({ 'data-kind': kind });
