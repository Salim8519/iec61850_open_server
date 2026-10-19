import { useEffect, useMemo, useState } from 'react';
import { api } from '../../api/client';
import type { Ied, Project, Signal } from '../../api/types';
import { useListFilter } from '../../state/useListFilter';
import { FilterBar } from '../../ui/FilterBar';
import { effective, type Choice } from '../../ui/filterState';
import { SIGNAL_FILTER, signalQuery } from './deviceFilters';
import { SignalTable } from './SignalTable';

const PAGE = 100;

/** The Data points tab of a simulated IED: the filter bar, then the table filling the height.
 *  Every filter runs in the backend over all points, so the count is exact (change 0093). */
export function DeviceSignals({ ied, project, values, perform, prepare }: {
  ied: Ied; project: Project; values: Record<string, unknown>;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
}) {
  const filter = useListFilter('data-points', SIGNAL_FILTER);
  const [offset, setOffset] = useState(0);
  const [signals, setSignals] = useState<Signal[]>([]);
  const [total, setTotal] = useState(0);
  const [devices, setDevices] = useState<string[]>([]);
  const [message, setMessage] = useState('');
  // True until the answer for the current IED and search arrives: never show "no match" too early.
  const [loading, setLoading] = useState(true);
  const running = project.runtime !== 'stopped';
  const options = useMemo<Record<string, Choice[]>>(() => ({
    ld: devices.map(name => ({ value: name, label: name })),
    report: ied.reports.map(item => ({ value: item.ref, label: `Report ${item.name}` })),
  }), [devices, ied.reports]);
  const chosen = useMemo(() => {
    const valid = effective(SIGNAL_FILTER, filter.values, options);
    return devices.length ? valid : { ...valid, ld: filter.values.ld };  // keep a remembered device until the list is known
  }, [filter.values, options, devices.length]);
  const query = signalQuery(chosen, offset);
  useEffect(() => { setOffset(0); }, [filter.values, ied.id]);
  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    const timer = setTimeout(() => api<{ items: Signal[]; total: number; devices?: string[] }>(`/ieds/${ied.id}/signals?${query}`)
      .then(result => { if (!cancelled) { setSignals(result.items); setTotal(result.total); setDevices(result.devices ?? []); } })
      .catch(reason => { if (!cancelled) setMessage(reason.message); })
      .finally(() => { if (!cancelled) setLoading(false); }), 150);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [ied.id, query, running]);
  const addTest = (signal: Signal) => perform(async () => {
    await prepare();
    await api('/project/test-panel', 'PUT', [...project.test_panel, { id: crypto.randomUUID(),
      label: (signal.description || signal.ref).slice(0, 100), ied_id: ied.id, ref: signal.ref, report_ref: null,
      duration_ms: 1000, active_value: true }]);
    setMessage(`Added “${signal.description || signal.ref}” to Test points.`);
  });
  return <>
    <FilterBar label="Filter data points" spec={SIGNAL_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={total} total={ied.signal_count} options={options} />
    {message && <p className="small muted panel-note" role="status">{message}</p>}
    <SignalTable signals={signals} values={values} iedId={ied.id} running={running} perform={perform} addTest={addTest}
      loading={loading} />
    <div className="pager"><span>{total ? offset + 1 : 0}–{Math.min(offset + PAGE, total)} of {total.toLocaleString()}</span>
      <div className="row"><button disabled={!offset} onClick={() => setOffset(Math.max(0, offset - PAGE))}>Previous</button>
        <button disabled={offset + PAGE >= total} onClick={() => setOffset(offset + PAGE)}>Next</button></div></div>
  </>;
}
