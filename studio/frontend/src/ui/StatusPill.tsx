import { Check, CircleAlert, Clock3, Info, Minus, X } from 'lucide-react';
import { TONES, type Status } from '../copy/status';
import './ui.css';

const ICONS = { check: Check, info: Info, clock: Clock3, alert: CircleAlert, x: X, minus: Minus };

/** Status as words + icon + border pattern; the colour only repeats what the words say. */
export function StatusPill({ status, showHint }: { status: Status; showHint?: boolean }) {
  const style = TONES[status.tone];
  const Icon = ICONS[style.icon];
  return <span className="status-wrap">
    <span className={`status-pill tone-${status.tone} pattern-${style.pattern}`} title={status.hint}>
      <Icon size={12} aria-hidden /><span className="sr-only">{style.word}: </span>{status.label}
    </span>
    {showHint && status.hint && <span className="status-hint">{status.hint}</span>}
  </span>;
}
