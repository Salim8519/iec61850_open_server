import { Check, CircleAlert, Clock3, X } from 'lucide-react';
import type { ValueStatus } from '../../../api/tools';

const ICON = { ok: Check, warning: CircleAlert, problem: X, error: X, pending: Clock3 };
const WORD = { ok: '', warning: 'Check: ', problem: 'Problem: ', error: 'Error: ', pending: 'Waiting: ' };

/** A value as text with an icon; the status word is read out, colour only repeats it. */
export function ValueText({ text, status }: { text: string; status: ValueStatus }) {
  const Icon = ICON[status];
  return <span className={`value-text value-${status}`}>
    <Icon size={13} aria-hidden /><span className="sr-only">{WORD[status]}</span>{text || '—'}
  </span>;
}

export const timeOf = (at: number | null) => at ? new Date(at * 1000).toLocaleTimeString() : '';
