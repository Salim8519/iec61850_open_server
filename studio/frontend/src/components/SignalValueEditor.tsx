import { useState } from 'react';
import type { Signal } from '../api/types';

export function SignalValueEditor({ signal, value, disabled, variant, update }: {
  signal: Signal; value: unknown; disabled: boolean; variant?: 'select' | 'segmented';
  update: (value: unknown) => void;
}) {
  const [draft, setDraft] = useState<string | null>(null);
  if (disabled || !signal.writable || signal.fc === 'CF') return <span className="mono">{String(value)}</span>;
  if (signal.kind === 'BOOLEAN' && variant === 'segmented')
    return <div className="segmented" role="group" aria-label={`Value ${signal.ref}`}>
      {[false, true].map(item => <button key={String(item)} type="button"
        className={value === item ? 'active' : ''} aria-pressed={value === item}
        onClick={() => update(item)}>{item ? 'True' : 'False'}</button>)}
    </div>;
  const options = signal.kind === 'BOOLEAN' ? { False: false, True: true } : signal.kind === 'Dbpos' ?
    { Intermediate: 0, Open: 1, Closed: 2, Invalid: 3 } : signal.enums;
  if (Object.keys(options).length) return <select aria-label={`Value ${signal.ref}`} value={String(value)}
    onChange={e => update(signal.kind === 'BOOLEAN' ? e.target.value === 'true' : Number(e.target.value))}>
    {!Object.values(options).some(item => item === value) && <option value={String(value)}>Current: {String(value)}</option>}
    {Object.entries(options).map(([label, item]) => <option key={label} value={String(item)}>{label} ({String(item)})</option>)}
  </select>;
  const numeric = signal.kind.startsWith('INT') || signal.kind.startsWith('FLOAT') ||
    ['Enum', 'Quality', 'Timestamp', 'Tcmd'].includes(signal.kind);
  const submit = () => {
    if (draft === null || (numeric && !draft.trim())) return;
    update(numeric ? Number(draft) : draft); setDraft(null);
  };
  return <div className="row"><input className="signal-input" aria-label={`Value ${signal.ref}`}
    type={numeric ? 'number' : 'text'} step={signal.kind.startsWith('INT') ? 1 : 'any'} value={draft ?? String(value)}
    onChange={e => setDraft(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') submit(); }} />
    <button disabled={draft === null || (numeric && !draft.trim())} onClick={submit}>Set</button></div>;
}
