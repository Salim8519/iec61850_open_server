import { Plus } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { GatewayPoint, PointKind, Protocol } from '../../../../api/gateway';
import { addApi, type Plan, type TypeEntry } from '../../../../api/gatewayAdd';
import { ADDRESS_LABEL } from '../../../../copy/gateway';
import { ADD_COPY } from '../../../../copy/gatewayAdd';
import { SaveProblem } from '../../../../ui/SaveStatus';
import { canAdd, problemLines, typedAddress } from './addPlan';
import { PlannedRows } from './PlannedRows';
import { TypeList } from './TypeList';

/** "Add a range": N made-up points of one type at consecutive addresses from X. The engine checks that every
 *  address is free; nothing is added while one is used. */
export function OwnRange({ protocol, rows, types, add, fail }: {
  protocol: Protocol; rows: GatewayPoint[]; types: TypeEntry[]; add: (points: GatewayPoint[]) => void;
  fail: (message: string) => void;
}) {
  const [kind, setKind] = useState<PointKind>('single');
  const [count, setCount] = useState('10');
  const [from, setFrom] = useState('');
  const [name, setName] = useState('');
  const [plan, setPlan] = useState<Plan | null>(null);
  const [problem, setProblem] = useState('');
  useEffect(() => { setPlan(null); }, [kind, count, from, name]);
  const show = () => {
    const howMany = typedAddress(count);
    const start = typedAddress(from);
    if (!howMany || start === undefined) { setProblem(`Type how many points and the ${ADDRESS_LABEL[protocol]} of the first one.`); return; }
    setProblem('');
    addApi.range(protocol, kind, howMany, start, name, rows).then(setPlan).catch(error => fail(error.message));
  };
  const problems = problemLines(plan);
  return <div className="add-steps">
    <p className="small muted">{ADD_COPY.rangeHint}</p>
    <TypeList types={types} value={kind} change={setKind} />
    <div className="row own-form-fields">
      <label>{ADD_COPY.count}<input className="gateway-number" inputMode="numeric" value={count}
        onChange={event => setCount(event.target.value)} /></label>
      <label>{ADD_COPY.from} ({ADDRESS_LABEL[protocol]})<input className="gateway-number" inputMode="numeric" value={from}
        onChange={event => setFrom(event.target.value)} /></label>
      <label>{ADD_COPY.namePrefix}<input value={name} placeholder={ADD_COPY.namePrefixPlaceholder} maxLength={100}
        onChange={event => setName(event.target.value)} /></label>
      <button type="button" onClick={show}>{ADD_COPY.planRange}</button>
    </div>
    <SaveProblem text={problem} />
    {plan && <PlannedRows plan={plan} protocol={protocol} />}
    {plan && <div className="row add-actions">
      <button type="button" className="primary" disabled={!canAdd(plan, false, false)} onClick={() => add(plan.points)}>
        <Plus size={16} />{ADD_COPY.add(plan.points.length)}</button>
      {problems.length > 0 && <span className="small add-blocked">⚠ {ADD_COPY.fixFirst}</span>}
    </div>}
  </div>;
}
