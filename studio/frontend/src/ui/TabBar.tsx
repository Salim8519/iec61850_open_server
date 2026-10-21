import type { ReactNode } from 'react';
import './ui.css';

/** The tabs of a panel; the chosen tab is underlined and bold, and announced as selected. `end`
 *  sits at the right end of the row (e.g. the Full view button, ADR-0095). */
export function TabBar<T extends string>({ tabs, value, change, label, end }: {
  tabs: [T, string][]; value: T; change: (tab: T) => void; label: string; end?: ReactNode;
}) {
  return <div className="tabbar" role="tablist" aria-label={label}>{tabs.map(([key, text]) =>
    <button key={key} type="button" role="tab" aria-selected={value === key} className={value === key ? 'active' : ''}
      onClick={() => change(key)}>{text}</button>)}{end && <span className="tabbar-end">{end}</span>}</div>;
}
