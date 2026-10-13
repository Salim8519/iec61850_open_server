import { Lightbulb } from 'lucide-react';
import { hint } from '../copy/hints';
import './ui.css';

/** A short plain-language explanation. Wording always comes from copy/hints.ts. */
export function Hint({ id, compact }: { id: string; compact?: boolean }) {
  const text = hint(id);
  if (!text) return null;
  return <p className={`guide-hint${compact ? ' compact' : ''}`}><Lightbulb size={13} aria-hidden />{text}</p>;
}
