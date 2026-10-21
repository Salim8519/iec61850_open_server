import { Pencil } from 'lucide-react';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { KIND_COPY } from '../../../copy/pointKinds';

export type EditChoice = { value: string; label: string };

/** A choice you can change in place (ADR-0095): the current choice is shown as it is (a type badge and its
 *  name) with the same ✎ pin as editable text; a click opens the list, picking one saves and closes.
 *  The caller owns saving; locked shows the plain value without the pin. */
export function ChoiceEdit({ shown, value, choices, onChange, label, disabled }: {
  shown: ReactNode; value: string; choices: EditChoice[]; onChange: (value: string) => void; label: string; disabled?: boolean;
}) {
  const [editing, setEditing] = useState(false);
  const list = useRef<HTMLSelectElement>(null);
  const pin = useRef<HTMLButtonElement>(null);
  // Open the list at once: the click that started editing is the user's gesture for it.
  useEffect(() => {
    if (!editing) return;
    try { list.current?.showPicker?.(); } catch { /* the list opens on the next click */ }
  }, [editing]);
  // After a pick or Esc the keyboard focus returns to the pin, so the next Tab continues in the row.
  const done = () => { setEditing(false); requestAnimationFrame(() => pin.current?.focus()); };
  if (disabled) return <span className="choice-locked">{shown}</span>;
  if (editing) return <select ref={list} autoFocus className="choice-select" value={value} aria-label={label}
    onChange={event => { done(); if (event.target.value !== value) onChange(event.target.value); }}
    onBlur={() => setEditing(false)} onKeyDown={event => { if (event.key === 'Escape') done(); }}>
    {choices.map(choice => <option key={choice.value} value={choice.value}>{choice.label}</option>)}</select>;
  return <button ref={pin} type="button" className="editable-text choice-shown" title={`${label}: ${KIND_COPY.edit}`}
    aria-label={`${label}: ${choices.find(choice => choice.value === value)?.label ?? value}`} onClick={() => setEditing(true)}>
    {shown}<Pencil className="edit-pin" size={12} aria-hidden /></button>;
}
