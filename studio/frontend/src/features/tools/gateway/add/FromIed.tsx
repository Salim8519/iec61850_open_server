import { Plus } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { GatewayPoint, PointKind, Protocol } from '../../../../api/gateway';
import { addApi, type Plan, type TypeEntry } from '../../../../api/gatewayAdd';
import type { Ied } from '../../../../api/types';
import { ADD_COPY } from '../../../../copy/gatewayAdd';
import { Spinner } from '../../../../ui/Spinner';
import { canAdd, picksOf, problemLines, typedAddress, type RowEdit } from './addPlan';
import { CandidateList } from './CandidateList';
import { IedChooser } from './IedChooser';
import { PlannedRows } from './PlannedRows';
import { RealWarning } from './RealWarning';
import { TypeList } from './TypeList';

/** "From an IED": IED → type → tick signals → names and next free addresses (editable) → Add N points.
 *  Commands are IED points, so SCADA commands on them operate that IED at once (the gateway command path);
 *  "Also add its status point" pairs each command with its status address. A real IED needs "I understand". */
export function FromIed({ protocol, ieds, rows, types, add, fail }: {
  protocol: Protocol; ieds: Ied[]; rows: GatewayPoint[]; types: TypeEntry[];
  add: (points: GatewayPoint[]) => void; fail: (message: string) => void;
}) {
  const [iedId, setIedId] = useState(ieds.length === 1 ? ieds[0].id : '');
  const [kind, setKind] = useState<PointKind | null>(null);
  const [ticked, setTicked] = useState<string[]>([]);
  const [edits, setEdits] = useState<Record<string, RowEdit>>({}); // typed addresses: the engine plans again
  const [names, setNames] = useState<Record<string, string>>({}); // typed names: kept here, no new plan needed
  const [withStatus, setWithStatus] = useState(true);
  const [plan, setPlan] = useState<Plan | null>(null);
  const [planning, setPlanning] = useState(false);
  const [understood, setUnderstood] = useState(false);
  const ied = ieds.find(item => item.id === iedId);
  const real = ied?.role === 'real';
  const command = !!kind?.startsWith('command');
  useEffect(() => { setTicked([]); setEdits({}); setNames({}); setPlan(null); }, [iedId, kind]);
  useEffect(() => setUnderstood(false), [iedId]);
  useEffect(() => {
    if (!ied || !kind || !ticked.length) { setPlan(null); return; }
    let live = true;
    setPlanning(true);
    const timer = setTimeout(() => addApi.fromIed(protocol, ied.id, kind, picksOf(ticked, edits), command && withStatus, rows)
      .then(answer => { if (live) setPlan(answer); }).catch(error => fail(error.message))
      .finally(() => { if (live) setPlanning(false); }), 150);
    return () => { live = false; clearTimeout(timer); };
  }, [ticked, edits, withStatus, iedId, kind]);
  const edit = (ref: string, change: RowEdit) => setEdits(current => ({ ...current, [ref]: { ...current[ref], ...change } }));
  const named = (point: GatewayPoint) => names[point.ref] !== undefined && ticked.includes(point.ref)
    ? { ...point, description: names[point.ref] } : point;
  const shown = plan && { ...plan, rows: plan.rows.map(row => row.status_for ? row : { ...row, point: named(row.point) }) };
  const count = plan?.points.length ?? 0;
  const problems = problemLines(plan);
  return <div className="add-steps">
    <section className="add-step"><h4>{ADD_COPY.step(1, ADD_COPY.chooseIed)}</h4>
      <IedChooser ieds={ieds} value={iedId} change={setIedId} /></section>
    {ied && <section className="add-step"><h4>{ADD_COPY.step(2, ADD_COPY.chooseType)}</h4>
      {types.length ? <TypeList types={types} value={kind} change={setKind} /> : <Spinner inline label="Loading the types…" />}</section>}
    {ied && kind && <section className="add-step"><h4>{ADD_COPY.step(3, ADD_COPY.chooseSignals)}</h4>
      <CandidateList protocol={protocol} ied={ied.id} kind={kind} rows={rows} ticked={ticked} setTicked={setTicked} fail={fail} />
      {command && <label className="row add-check" title={ADD_COPY.withStatusHint(protocol)}>
        <input type="checkbox" checked={withStatus} onChange={event => setWithStatus(event.target.checked)} />
        <span>{ADD_COPY.withStatus}<small className="muted"> · {ADD_COPY.withStatusHint(protocol)}</small></span></label>}
    </section>}
    {plan && <section className="add-step"><h4>{ADD_COPY.step(4, ADD_COPY.rowsTitle)}</h4>
      <p className="small muted">{ADD_COPY.rowsHint}</p>
      <PlannedRows plan={shown!} protocol={protocol} editing={{
        name: (row, text) => setNames(current => ({ ...current, [row.point.ref]: text })),
        address: (row, text) => { const typed = typedAddress(text); if (typed !== row.point.address) edit(row.point.ref, { address: typed }); },
      }} /></section>}
    {plan && real && <RealWarning understood={understood} change={setUnderstood} />}
    {plan && <div className="row add-actions">
      <button type="button" className="primary" disabled={planning || !canAdd(plan, real, understood)} onClick={() => add(plan.points.map(named))}>
        <Plus size={16} />{ADD_COPY.add(count)}</button>
      {planning && <Spinner inline size={14} label={ADD_COPY.check} />}
      {problems.length > 0 && <span className="small add-blocked">⚠ {ADD_COPY.fixFirst}</span>}
    </div>}
  </div>;
}
