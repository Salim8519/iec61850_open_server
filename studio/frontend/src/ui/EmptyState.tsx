import type { ReactNode } from 'react';
import { hint } from '../copy/hints';
import './ui.css';

/** Says what this area is for and offers the one next step. */
export function EmptyState({ icon, title, hintId, action }: {
  icon: ReactNode; title: string; hintId: string; action?: ReactNode;
}) {
  return <div className="guide-empty">
    <span className="guide-empty-icon">{icon}</span>
    <h3>{title}</h3>
    <p>{hint(hintId)}</p>
    {action}
  </div>;
}
