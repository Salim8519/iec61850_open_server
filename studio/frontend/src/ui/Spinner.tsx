import { Loader2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import './ui.css';

/** Shown after this long, so a wait never looks frozen. */
const SLOW_MS = 8000;

/** The one waiting indicator: a turning ring and words saying what is happening.
 *  After a few seconds it adds a calm "still working" line instead of looking stuck. */
export function Spinner({ label, size = 16, slowHint, inline = false }: {
  label?: string; size?: number; slowHint?: string; inline?: boolean;
}) {
  const [slow, setSlow] = useState(false);
  useEffect(() => {
    if (!slowHint) return;
    const timer = window.setTimeout(() => setSlow(true), SLOW_MS);
    return () => window.clearTimeout(timer);
  }, [slowHint]);
  return <span className={`ui-spinner${inline ? ' inline' : ''}`} role="status" aria-live="polite">
    <Loader2 size={size} className="ui-spin" aria-hidden="true" />
    {label && <span>{label}</span>}
    {slow && slowHint && <span className="ui-spinner-slow">{slowHint}</span>}
  </span>;
}

/** The turning ring alone, for inside a busy button next to its own words. */
export function SpinIcon({ size = 16 }: { size?: number }) {
  return <Loader2 size={size} className="ui-spin" aria-hidden="true" />;
}
