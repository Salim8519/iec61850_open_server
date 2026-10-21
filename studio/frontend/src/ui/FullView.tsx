import { Maximize2, Minimize2 } from 'lucide-react';
import { useEffect, type ReactNode } from 'react';
import { FULL_VIEW_COPY } from '../copy/fullView';
import './full-view.css';

/** The "⤢ Full view" button at the right end of a list's filter bar (ADR-0095). */
export function FullViewButton({ full, change }: { full: boolean; change: (on: boolean) => void }) {
  if (full) return null;
  return <button type="button" className="full-view-button" title={FULL_VIEW_COPY.openHint} onClick={() => change(true)}>
    <Maximize2 size={15} aria-hidden />{FULL_VIEW_COPY.open}
  </button>;
}

/** Wraps one big list. In full view the list covers the whole window: the app sidebar, top bar,
 *  notices, tool header and side columns are out of sight, while every button, filter, column and
 *  page control of the list stays. A slim bar says where you are, can show the list's own state
 *  (`status`, e.g. "Gateway stopped" and its Start button) and leaves full view; so does Esc. */
export function FullView({ full, change, place, detail, status, children }: {
  full: boolean; change: (on: boolean) => void; place: string; detail?: string; status?: ReactNode; children: ReactNode;
}) {
  useEffect(() => {
    if (!full) return;
    const key = (event: KeyboardEvent) => {
      if (event.key !== 'Escape' || event.defaultPrevented) return;
      const target = event.target as HTMLElement | null;
      if (target?.closest('input, textarea, select, [role=dialog], [role=alertdialog], .popover')) return;
      change(false);
    };
    window.addEventListener('keydown', key);
    return () => window.removeEventListener('keydown', key);
  }, [full, change]);
  if (!full) return <>{children}</>;
  return <div className="full-view" role="region" aria-label={`${place} ${FULL_VIEW_COPY.region}`}>
    <div className="full-view-bar">
      <strong>{place}</strong>{detail && <span className="full-view-detail">{detail}</span>}
      <span className="full-view-gap" />
      {status}
      <button type="button" className="full-view-exit" title={FULL_VIEW_COPY.exitHint} onClick={() => change(false)}>
        <Minimize2 size={15} aria-hidden />{FULL_VIEW_COPY.exit}
      </button>
    </div>
    <div className="full-view-body">{children}</div>
  </div>;
}
