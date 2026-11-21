import './ui.css';

/** A few mutually exclusive choices shown side by side, like iOS. One choice can be locked on its own
 *  (disabled) and can carry a green "running" dot (live) next to its words. */
export function SegmentedControl<T extends string>({ value, options, onChange, label, disabled, full }: {
  value: T; options: { value: T; label: string; title?: string; disabled?: boolean; live?: boolean }[];
  onChange: (value: T) => void;
  label: string; disabled?: boolean; full?: boolean;
}) {
  return <div className={`segmented-control${full ? ' full' : ''}`} role="radiogroup" aria-label={label}>
    {options.map(option => <button key={option.value} type="button" role="radio" aria-checked={value === option.value}
      className={value === option.value ? 'active' : ''} disabled={disabled || option.disabled}
      title={option.title} onClick={() => onChange(option.value)}>{option.label}
      {option.live && <><span className="live-dot segment-live" aria-hidden /><span className="sr-only"> (running)</span></>}</button>)}
  </div>;
}
