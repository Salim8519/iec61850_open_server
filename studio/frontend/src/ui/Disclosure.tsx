import { ChevronRight } from 'lucide-react';
import type { ReactNode } from 'react';
import './ui.css';

/** The single place for rarely needed options. Closed by default. */
export function Disclosure({ title = 'Advanced', children }: { title?: string; children: ReactNode }) {
  return <details className="disclosure">
    <summary><ChevronRight size={16} aria-hidden />{title}</summary>
    <div className="disclosure-body">{children}</div>
  </details>;
}
