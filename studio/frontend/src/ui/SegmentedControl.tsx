import './ui.css';

/** A few mutually exclusive choices shown side by side, like iOS. */
export function SegmentedControl<T extends string>({ value, options, onChange, label, disabled, full }: {
  value: T; options: { value: T; label: string; title?: string }[]; onChange: (value: T) => void;
  label: string; disabled?: boolean; full?: boolean;
}) {
  return <div className={`segmented-control${full ? ' full' : ''}`} role="radiogroup" aria-label={label}>
    {options.map(option => <button key={option.value} type="button" role="radio" aria-checked={value === option.value}
      className={value === option.value ? 'active' : ''} disabled={disabled} title={option.title}
      onClick={() => onChange(option.value)}>{option.label}</button>)}
  </div>;
}
