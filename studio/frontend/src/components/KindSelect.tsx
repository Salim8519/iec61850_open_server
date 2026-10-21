import { KIND_COPY, KIND_TEXT } from '../copy/pointKinds';
import { KINDS, type Kind } from '../points/pointKind';
import { ALL_KINDS } from '../ui/kindFilter';

/** The point-type filter for narrow lists (SLD sheets, pickers): one dropdown instead of a chip per kind.
 *  With `counts` it lists only the kinds the list has; otherwise every kind (ADR-0095). */
export function KindSelect({ value, onChange, counts, disabled }: {
  value: string; onChange: (kind: string) => void; counts?: Partial<Record<Kind, number>>; disabled?: boolean;
}) {
  const shown = KINDS.filter(kind => !counts || counts[kind] || kind === value);
  return <select className={`kind-select${value !== ALL_KINDS ? ' active' : ''}`} aria-label={KIND_COPY.chips}
    title={KIND_COPY.chips} value={value} disabled={disabled} onChange={event => onChange(event.target.value)}>
    <option value={ALL_KINDS}>{KIND_COPY.all}</option>
    {shown.map(kind => <option key={kind} value={kind}>
      {KIND_TEXT[kind].glyph} {kind} · {KIND_TEXT[kind].name}{counts?.[kind] ? ` (${counts[kind]})` : ''}</option>)}
  </select>;
}
