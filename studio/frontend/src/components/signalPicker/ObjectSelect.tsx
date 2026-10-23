import { PICKER_COPY } from '../../copy/signalPicker';
import { objectOptions } from './pickerAnswer';

/** The object (logical node) filter of a picker (ADR-0113): "All objects" and every node with its count
 *  from the engine ("ConInGGIO1 (200)"), in natural order. */
export function ObjectSelect({ counts, value, total, disabled, onChange }: {
  counts: Record<string, number>; value: string; total: number; disabled?: boolean; onChange: (ln: string) => void;
}) {
  return <select aria-label={PICKER_COPY.objects} title={PICKER_COPY.objectsHint} value={value} disabled={disabled}
    className={value ? 'active' : ''} onChange={event => onChange(event.target.value)}>
    <option value="">{PICKER_COPY.allObjects} ({total.toLocaleString()})</option>
    {objectOptions(counts, value).map(option =>
      <option key={option.value} value={option.value}>{option.label} ({option.count.toLocaleString()})</option>)}
  </select>;
}
