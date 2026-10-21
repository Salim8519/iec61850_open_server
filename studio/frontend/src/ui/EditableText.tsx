import { Pencil, RefreshCw } from 'lucide-react';
import { useState, type KeyboardEvent } from 'react';
import { KIND_COPY } from '../copy/pointKinds';
import './kind.css';

/** Text you can change in place (ADR-0095): shows the value with a small ✎ pin and a dashed line;
 *  a click turns it into an input. Enter or leaving the field saves, Esc cancels. The caller owns
 *  saving; this control only reports the new text. */
export function EditableText({ value, onSave, label, placeholder, disabled, mono, inputMode }: {
  value: string; onSave: (text: string) => void; label: string; placeholder?: string; disabled?: boolean;
  mono?: boolean; inputMode?: 'text' | 'decimal' | 'numeric';
}) {
  const [draft, setDraft] = useState<string | null>(null);
  if (disabled) return <span className={`editable-text locked${mono ? ' mono' : ''}`}>{value || placeholder}</span>;
  if (draft !== null) {
    const finish = (save: boolean) => { if (save && draft !== value) onSave(draft); setDraft(null); };
    const key = (event: KeyboardEvent<HTMLInputElement>) => {
      if (event.key === 'Enter') finish(true);
      if (event.key === 'Escape') finish(false);
    };
    return <input className={`editable-input${mono ? ' mono' : ''}`} autoFocus value={draft} aria-label={label}
      inputMode={inputMode} title={KIND_COPY.save} onChange={event => setDraft(event.target.value)}
      onKeyDown={key} onBlur={() => finish(true)} />;
  }
  return <button type="button" className={`editable-text${value ? '' : ' is-empty'}${mono ? ' mono' : ''}`}
    title={`${label}: ${KIND_COPY.edit}`} aria-label={`${label}: ${value || placeholder || ''}`} onClick={() => setDraft(value)}>
    <span>{value || placeholder}</span><Pencil className="edit-pin" size={12} aria-hidden />
  </button>;
}

/** A value that changes by itself: the value plus a dotted "auto" tag instead of the ✎ pin. */
export function AutoValue({ value, how }: { value: string; how: string }) {
  return <span className="auto-value" title={KIND_COPY.autoHint(how)}>
    <span>{value}</span><span className="auto-tag"><RefreshCw size={11} aria-hidden />{KIND_COPY.auto}</span>
  </span>;
}
