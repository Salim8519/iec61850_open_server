import { ListChecks, Plus, X } from 'lucide-react';
import { useMemo, useState } from 'react';
import { api } from '../../api/client';
import type { Project, TestControl } from '../../api/types';
import { SignalSearchPicker } from '../../components/SignalSearchPicker';
import { useDataClassesOf } from '../../components/useDataClassesOf';
import '../../components/signal-search-picker.css';
import { countKinds } from '../../points/pointKind';
import { useListFilter } from '../../state/useListFilter';
import { EmptyState } from '../../ui/EmptyState';
import { FilterBar } from '../../ui/FilterBar';
import { effective, isFiltered } from '../../ui/filterState';
import { Spinner } from '../../ui/Spinner';
import { TestPointRow, type Item } from './TestPointRow';
import { swapPoints, TEST_FILTER, testIedChoices, testPointKind, testPointMatches } from './testFilters';
import { useLiveFeed } from '../../api/useLiveFeed';

const STOP_TO_ADD = 'Stop the run to add test points';
/** Below this many points a filter bar is only noise; it appears at once if a remembered filter is on. */
const FILTER_FROM = 6;

type Perform = (task: () => Promise<unknown>) => Promise<void>;

export function TestPoints({ project, perform, prepare }: {
  project: Project; perform: Perform; prepare: () => Promise<void>;
}) {
  const [adding, setAdding] = useState(false);
  const filter = useListFilter('test-points', TEST_FILTER);
  const running = project.runtime !== 'stopped';
  const entries = project.test_panel || [];
  // Shared live feed: every second while running, once otherwise; unchanged answers do nothing.
  const panel = useLiveFeed<{ items: Item[] }>('/project/test-panel', running ? 1000 : null, project.test_panel).value;
  const items = panel?.items ?? [];
  const loaded = !!panel;
  const classFor = useDataClassesOf(items.map(item => item.ied_id));
  const options = useMemo(() => ({ ied: testIedChoices(items, project.ieds) }), [items, project.ieds]);
  const values = effective(TEST_FILTER, filter.values, options);
  const rows = items.map(item => {
    const ied = project.ieds.find(entry => entry.id === item.ied_id);
    return { item, ied, kind: testPointKind(item, ied, ref => classFor(item.ied_id, ref)) };
  });
  const inScope = rows.filter(row => testPointMatches(row.item, row.kind, row.ied?.name, { ...values, point_kind: 'all' }));
  const shown = rows.filter(row => testPointMatches(row.item, row.kind, row.ied?.name, values));
  const save = (next: TestControl[]) => perform(async () => { await prepare(); await api('/project/test-panel', 'PUT', next); });
  const showBar = items.length >= FILTER_FROM || isFiltered(TEST_FILTER, filter.values);
  return <div className="stack">
    <div className="row test-points-head">
      <p className="muted">{running ? 'Your saved alarms, statuses and measurements. Change them here while the IEDs run; stop the run to add or remove test points.'
        : 'Your saved alarms, statuses and measurements. Press Run, then change them here.'}</p>
      {(adding || !!items.length) && <button className={adding ? '' : 'primary'} disabled={running} title={running ? STOP_TO_ADD : undefined}
        onClick={() => setAdding(!adding)}>
        {adding ? <><X size={16} />Close</> : <><Plus size={16} />Add test point</>}</button>}
    </div>
    {adding && <div className="card"><div className="card-body"><SignalSearchPicker ieds={project.ieds} writableOnly chooseLabel="Add" memoryKey="test-points"
      exclude={new Set(entries.map(item => `${item.ied_id}:${item.ref}`))}
      choose={(binding, signal) => save([...entries, { id: crypto.randomUUID(), label: (signal.description || signal.ref).slice(0, 100),
        ...binding, duration_ms: 1000, active_value: true }])} /></div></div>}
    {!loaded && !items.length && <div className="card"><div className="card-body"><Spinner label="Loading your test points…" /></div></div>}
    {loaded && !items.length && !adding && <div className="card"><EmptyState icon={<ListChecks size={24} />} title="No test points yet"
      hintId="test.points.empty" action={<button className="primary" disabled={running} title={running ? STOP_TO_ADD : undefined}
        onClick={() => setAdding(true)}>
        <Plus size={16} />Add test point</button>} /></div>}
    {!!items.length && <div className="card">
      {showBar && <FilterBar label="Filter test points" spec={TEST_FILTER} values={values} set={filter.set} clear={filter.clear}
        shown={shown.length} total={items.length} options={options} counts={{ point_kind: countKinds(inScope, row => row.kind) }} />}
      <table className="compact-rows"><thead><tr><th title="Hover a badge for its full name and the exact data type">Type</th><th>Name</th><th>Value</th><th style={{ width: 120 }} /></tr></thead>
        <tbody>{shown.map(({ item, ied, kind }, index) => {
          const entry = entries.find(candidate => candidate.id === item.id);
          const above = shown[index - 1]?.item.id;
          const below = shown[index + 1]?.item.id;
          return entry && <TestPointRow key={item.id} item={item} entry={entry} kind={kind} iedName={ied?.name}
            running={running} perform={perform}
            edit={next => save(entries.map(current => current.id === next.id ? next : current))}
            move={{ up: above ? () => save(swapPoints(entries, item.id, above)) : undefined,
              down: below ? () => save(swapPoints(entries, item.id, below)) : undefined }}
            remove={() => save(entries.filter(current => current.id !== item.id))} />;
        })}</tbody></table>
      {!shown.length && <div className="empty">No test point matches. Press Clear to see them all.</div>}</div>}
  </div>;
}
