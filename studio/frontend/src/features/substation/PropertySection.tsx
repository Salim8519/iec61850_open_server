import type { ReactNode } from 'react';
import './property-sections.css';

export type SectionTone = 'neutral' | 'ied' | 'command' | 'live' | 'legacy';

const TONE_BADGE: Record<SectionTone, string | null> = {
  neutral: null, ied: 'Reads from IED', command: 'Sends command', live: 'Live', legacy: 'Legacy',
};

/** Bordered property card: tinted title band, domain tag in words, optional hint. */
export function PropertySection({ tone, title, badge, hint, children }: {
  tone: SectionTone; title: string; badge?: string; hint?: string; children: ReactNode;
}) {
  const tag = badge || TONE_BADGE[tone];
  return <section className={`prop-card tone-${tone}`} aria-label={title}>
    <header className="prop-heading">
      <span className="prop-dot" aria-hidden="true" />
      <h3>{title}</h3>
      {tag && <span className="prop-tag">{tag}</span>}
    </header>
    <div className="prop-body">
      {hint && <p className="prop-hint">{hint}</p>}
      {children}
    </div>
  </section>;
}
