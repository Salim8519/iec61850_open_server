import { Check } from 'lucide-react';
import type { ReactNode } from 'react';
import { PICKER_COPY } from '../../copy/signalPicker';
import { KIND_TEXT } from '../../copy/pointKinds';
import type { Kind } from '../../points/pointKind';
import { KindBadge } from '../../ui/KindBadge';
import { ALL_KINDS } from '../../ui/kindFilter';
import { allCount, kindChips } from './pickerAnswer';
import '../../ui/filter-bar.css';

/** The type chips of a picker (ADR-0113): "All types" and one chip per type that has rows, each with the
 *  shared badge (letters, symbol, colour) and its count from the engine. The chosen chip has a tick and a
 *  solid border, never colour alone. `why` says why a type was chosen for the user. */
export function KindChips({ counts, value, why, disabled, onChange }: {
  counts: Partial<Record<Kind, number>>; value: string; why?: string; disabled?: boolean;
  onChange: (kind: string) => void;
}) {
  const chip = (kind: string, label: ReactNode, count: number, title: string) => {
    const active = value === kind;
    return <button key={kind} type="button" role="radio" aria-checked={active} disabled={disabled} title={title}
      className={`chip picker-chip${active ? ' active' : ''}`} onClick={() => onChange(kind)}>
      {active && <Check size={13} aria-hidden />}{label}<span className="chip-count">{count.toLocaleString()}</span>
    </button>;
  };
  return <div className="picker-kinds">
    <div className="chip-group picker-chips" role="radiogroup" aria-label={PICKER_COPY.types} title={PICKER_COPY.typesHint}>
      {chip(ALL_KINDS, PICKER_COPY.allTypes, allCount(counts), PICKER_COPY.typesHint)}
      {kindChips(counts, value).map(({ kind, count }) => chip(kind,
        <><KindBadge kind={kind} /><span className="picker-chip-name">{KIND_TEXT[kind].name}</span></>,
        count, `${KIND_TEXT[kind].name}: ${KIND_TEXT[kind].means}`))}
    </div>
    {why && <p className="small muted picker-why" role="note">{why}</p>}
  </div>;
}
