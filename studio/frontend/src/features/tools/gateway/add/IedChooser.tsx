import { Search } from 'lucide-react';
import { useState } from 'react';
import type { Ied } from '../../../../api/types';
import { ADD_COPY, iedBadge } from '../../../../copy/gatewayAdd';
import { StatusPill } from '../../../../ui/StatusPill';
import { filterIeds } from './addPlan';

/** Step 1 of "From an IED": a searchable list of the project IEDs, each with its Simulated / Real badge. */
export function IedChooser({ ieds, value, change }: { ieds: Ied[]; value: string; change: (id: string) => void }) {
  const [query, setQuery] = useState('');
  if (!ieds.length) return <p className="small muted">{ADD_COPY.noIeds}</p>;
  const shown = filterIeds(ieds, query);
  return <div className="ied-chooser">
    {ieds.length > 6 && <div className="search-field"><Search size={14} aria-hidden />
      <input aria-label={ADD_COPY.searchIeds} placeholder={ADD_COPY.searchIeds} value={query}
        onChange={event => setQuery(event.target.value)} /></div>}
    <div className="ied-choices" role="radiogroup" aria-label={ADD_COPY.chooseIed}>
      {shown.map(ied => <button key={ied.id} type="button" role="radio" aria-checked={value === ied.id}
        className={`ied-choice${value === ied.id ? ' active' : ''}`} onClick={() => change(ied.id)}>
        <strong>{ied.name}</strong><StatusPill status={iedBadge(ied.role)} /></button>)}
      {!shown.length && <p className="small muted">{ADD_COPY.noIedMatch}</p>}
    </div>
  </div>;
}
