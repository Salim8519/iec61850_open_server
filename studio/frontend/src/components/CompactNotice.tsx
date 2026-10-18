import { ChevronDown, ChevronUp, Info, TriangleAlert, X } from 'lucide-react';
import { useState, type ReactNode } from 'react';

/** A one-line notice above the page (change 0093): the message, a Details button and Dismiss.
 *  Details open over the page instead of pushing it down, so lists stay where they are. */
export function CompactNotice({ message, problem, detailsLabel, dismissLabel, children }: {
  message: string; problem?: boolean; detailsLabel: string; dismissLabel: string; children?: ReactNode;
}) {
  const [dismissed, setDismissed] = useState(false);
  const [open, setOpen] = useState(false);
  if (dismissed) return null;
  const Icon = problem ? TriangleAlert : Info;
  return <div className={`notice compact${problem ? ' problem' : ''}`} role={problem ? 'alert' : 'status'}>
    <Icon size={16} aria-hidden className="notice-icon" />
    <span className="notice-text" title={message}>{message}</span>
    {children && <button className="ghost" aria-expanded={open} onClick={() => setOpen(!open)}>
      {open ? <ChevronUp size={15} /> : <ChevronDown size={15} />}{detailsLabel}</button>}
    <button className="ghost" aria-label={dismissLabel} title="Dismiss" onClick={() => setDismissed(true)}><X size={16} /></button>
    {open && children && <div className="notice-details" role="region" aria-label={detailsLabel}>{children}</div>}
  </div>;
}
