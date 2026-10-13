import { CircleHelp } from 'lucide-react';
import { useId, useState } from 'react';
import { glossaryTerm } from '../copy/glossary';
import './ui.css';

/** The term with a "?" that explains it; works with mouse, keyboard and touch. */
export function InfoTip({ term, label }: { term: string; label?: string }) {
  const entry = glossaryTerm(term);
  const [open, setOpen] = useState(false);
  const id = useId();
  if (!entry) return <>{label}</>;
  return <span className="info-tip" onMouseLeave={() => setOpen(false)}>
    {label ?? entry.term}
    <button type="button" className="info-tip-button" aria-label={`What is ${entry.term}?`}
      aria-expanded={open} aria-describedby={open ? id : undefined}
      onClick={() => setOpen(!open)} onMouseEnter={() => setOpen(true)} onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}>
      <CircleHelp size={13} />
    </button>
    {open && <span role="tooltip" id={id} className="info-tip-bubble"><strong>{entry.term}</strong>{entry.plain}</span>}
  </span>;
}
