import { Check, Search, X } from 'lucide-react';
import type { ReactNode } from 'react';
import { countText, isFiltered, type Choice, type FilterSpec, type FilterValues } from './filterState';
import './filter-bar.css';

/** The one filter bar of every list: search, choice chips, toggles, dropdowns and "N of M · Clear".
 *  It only shows and changes filter values; each list decides which rows match (change 0093). */
export function FilterBar({ spec, values, set, clear, shown, total, counts, options, extra, label, note }: {
  spec: FilterSpec; values: FilterValues; set: (key: string, value: string | boolean) => void; clear: () => void;
  shown: number; total: number; label: string; counts?: Record<string, Record<string, number>>;
  options?: Record<string, Choice[]>; extra?: ReactNode; note?: string;
}) {
  const filtered = isFiltered(spec, values);
  return <div className="filter-bar" role="search" aria-label={label}>
    {spec.search !== undefined && <label className="search-field filter-search"><Search size={16} aria-hidden />
      <input value={String(values.text ?? '')} placeholder={spec.search} aria-label={spec.search}
        onChange={event => set('text', event.target.value)} spellCheck={false} /></label>}
    {spec.groups?.map(group => <span key={group.key} className="chip-group" role="radiogroup" aria-label={group.label}>
      {group.options.map(option => {
        const active = values[group.key] === option.value;
        const count = counts?.[group.key]?.[option.value];
        return <button key={option.value} type="button" role="radio" aria-checked={active}
          className={`chip${active ? ' active' : ''}`} onClick={() => set(group.key, option.value)}>
          {active && <Check size={13} aria-hidden />}{option.label}{count !== undefined && <span className="chip-count">{count}</span>}
        </button>;
      })}</span>)}
    {spec.toggles?.map(toggle => {
      const active = Boolean(values[toggle.key]);
      return <button key={toggle.key} type="button" role="checkbox" aria-checked={active} title={toggle.hint}
        className={`chip toggle${active ? ' active' : ''}`} onClick={() => set(toggle.key, !active)}>
        <span className="chip-box" aria-hidden>{active && <Check size={12} />}</span>{toggle.label}
      </button>;
    })}
    {spec.selects?.map(select => {
      const list = options?.[select.key] ?? [];
      if (list.length < 2 && !values[select.key]) return null;
      return <select key={select.key} className={`filter-select${values[select.key] ? ' active' : ''}`} aria-label={select.label}
        value={list.some(option => option.value === values[select.key]) ? String(values[select.key]) : ''}
        onChange={event => set(select.key, event.target.value)}>
        <option value="">{select.all}</option>
        {list.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
      </select>;
    })}
    <span className="filter-spacer" />
    {extra}
    <span className="filter-count" aria-live="polite">{countText(shown, total, filtered)}{note ? ` ${note}` : ''}</span>
    {filtered && <button type="button" className="ghost filter-clear" onClick={clear}><X size={14} />Clear</button>}
  </div>;
}
