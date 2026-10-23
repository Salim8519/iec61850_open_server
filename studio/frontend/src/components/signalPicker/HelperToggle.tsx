import { Check } from 'lucide-react';
import { PICKER_COPY } from '../../copy/signalPicker';
import '../../ui/filter-bar.css';

/** "Show quality and time": the q and t helper attributes are hidden until this is ticked (ADR-0113).
 *  Off is a dashed chip, on a solid one with a tick, like every toggle of the filter bar. */
export function HelperToggle({ shown, disabled, onChange }: {
  shown: boolean; disabled?: boolean; onChange: (shown: boolean) => void;
}) {
  return <button type="button" role="checkbox" aria-checked={shown} disabled={disabled} title={PICKER_COPY.helpersHint}
    className={`chip toggle picker-helpers${shown ? ' active' : ''}`} onClick={() => onChange(!shown)}>
    <span className="chip-box" aria-hidden>{shown && <Check size={12} />}</span>{PICKER_COPY.helpers}
  </button>;
}
