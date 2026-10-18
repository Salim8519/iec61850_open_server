import './ui.css';

/** The tabs of a panel; the chosen tab is underlined and bold, and announced as selected. */
export function TabBar<T extends string>({ tabs, value, change, label }: {
  tabs: [T, string][]; value: T; change: (tab: T) => void; label: string;
}) {
  return <div className="tabbar" role="tablist" aria-label={label}>{tabs.map(([key, text]) =>
    <button key={key} type="button" role="tab" aria-selected={value === key} className={value === key ? 'active' : ''}
      onClick={() => change(key)}>{text}</button>)}</div>;
}
