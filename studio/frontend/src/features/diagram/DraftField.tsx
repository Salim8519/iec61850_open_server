import { useEffect, useState } from 'react';

type Props = {
  label: string; value: string; disabled?: boolean; inputMode?: 'decimal' | 'numeric' | 'text';
  maxLength?: number; placeholder?: string; title?: string;
  /** Returns an error sentence to keep the typed text, or '' when the value was taken. */
  commit: (text: string) => string;
  /** Tells the parent the current error ('' when fixed), so it can show one line below the fields. */
  report?: (error: string) => void;
};

/** Small labelled field that keeps half-typed text (such as "0.") and saves on Enter or leaving it. */
export function DraftField({ label, value, disabled, inputMode = 'text', maxLength, placeholder, title,
  commit, report }: Props) {
  const [text, setText] = useState(value);
  const [invalid, setInvalid] = useState(false);
  useEffect(() => { setText(value); setInvalid(false); }, [value]);
  const settle = (error: string) => { setInvalid(!!error); report?.(error); };
  const save = () => settle(text === value ? '' : commit(text));
  return <label className={`draft-field${invalid ? ' invalid' : ''}`} title={title}>
    <span>{label}</span>
    <input value={text} disabled={disabled} inputMode={inputMode} maxLength={maxLength} placeholder={placeholder}
      aria-invalid={invalid} onChange={event => setText(event.target.value)} onBlur={save}
      onKeyDown={event => {
        if (event.key === 'Enter') (event.target as HTMLInputElement).blur();
        if (event.key === 'Escape') { setText(value); settle(''); }
      }} />
  </label>;
}
