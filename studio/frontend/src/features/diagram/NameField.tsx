import { useEffect, useState } from 'react';
import './relay-signals.css';

/** The equipment's name: each typed change shows on the diagram at once, but an empty name is never
 *  sent (the project refuses it). The field may be empty while typing; leaving it empty brings the
 *  saved name back. */
export function NameField({ value, disabled, change }: {
  value: string; disabled: boolean; change: (name: string) => void;
}) {
  const [text, setText] = useState(value);
  useEffect(() => setText(value), [value]);
  const empty = !text.trim();
  return <>
    <label>Name<input value={text} disabled={disabled} maxLength={100} aria-invalid={empty}
      onChange={event => { setText(event.target.value); if (event.target.value.trim()) change(event.target.value); }}
      onBlur={() => { if (empty) setText(value); }} /></label>
    {empty && <p className="relay-error" role="alert">Type a name: it cannot be empty.</p>}
  </>;
}
