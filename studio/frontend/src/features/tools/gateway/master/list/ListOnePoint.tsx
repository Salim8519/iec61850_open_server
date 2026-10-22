import { Plus } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { MasterProtocol } from '../../../../../api/master';
import { masterListApi, type ListPlan, type ListPoint, type MasterType } from '../../../../../api/masterList';
import { ADD_LIST_COPY, LIST_ADDRESS } from '../../../../../copy/masterList';
import { SaveProblem } from '../../../../../ui/SaveStatus';
import { TypeList } from '../../add/TypeList';
import { typedAddress } from '../../add/addPlan';
import { kindOfListType, planProblems, planReady, typedNumber } from './listRows';
import { ListPlanRows } from './ListPlanRows';

/** "One point" or "A range": a type from the engine's list, the address the device uses and a name.
 *  The engine checks the address against the list before anything is added. */
export function ListOnePoint({ protocol, types, range, add }: {
  protocol: MasterProtocol; types: MasterType[]; range: boolean; add: (points: ListPoint[]) => void;
}) {
  const [kind, setKind] = useState<string>(types[0]?.kind ?? '');
  const [address, setAddress] = useState('');
  const [name, setName] = useState('');
  const [count, setCount] = useState('10');
  const [scale, setScale] = useState('1');
  const [unit, setUnit] = useState('');
  const [plan, setPlan] = useState<ListPlan | null>(null);
  const [problem, setProblem] = useState('');
  useEffect(() => { if (!kind && types.length) setKind(types[0].kind); }, [types]);
  useEffect(() => setPlan(null), [kind, address, name, count, scale, unit]);
  const byKind = new Map(types.map(type => [type.kind, type]));
  const chosen = byKind.get(kind);
  const check = () => {
    const at = typedAddress(address);
    const howMany = typedAddress(count);
    const factor = typedNumber(scale);
    if (at === undefined || (range && !howMany)) { setProblem(range ? 'Type how many points and the first address.' : 'Type the address the device uses.'); return; }
    if (factor === undefined || factor === 0) { setProblem('The scale is a number other than 0.'); return; }
    setProblem('');
    const ask = range ? masterListApi.range(kind, howMany!, at, name.trim())
      : masterListApi.one({ type: kind, address: at, name: name.trim(), scale: factor, unit: unit.trim() });
    ask.then(setPlan).catch(error => setProblem(error.message));
  };
  const problems = planProblems(plan);
  return <div className="add-steps">
    <TypeList types={types} value={kind} change={setKind} label={ADD_LIST_COPY.typeChoose}
      badge={key => kindOfListType(byKind.get(key))} />
    <div className="row own-form-fields">
      {range && <label>{ADD_LIST_COPY.count}<input className="gateway-number" inputMode="numeric" value={count}
        onChange={event => setCount(event.target.value)} /></label>}
      <label>{range ? `${ADD_LIST_COPY.from} (${LIST_ADDRESS[protocol]})` : LIST_ADDRESS[protocol]}
        <input className="gateway-number" inputMode="numeric" value={address} onChange={event => setAddress(event.target.value)} /></label>
      <label>{range ? ADD_LIST_COPY.namePrefix : ADD_LIST_COPY.name}<input value={name} maxLength={100}
        placeholder={range ? ADD_LIST_COPY.namePrefixPlaceholder : ''} onChange={event => setName(event.target.value)} /></label>
      {!range && chosen && !chosen.command && <>
        <label>{ADD_LIST_COPY.scale}<input className="gateway-number" inputMode="decimal" value={scale}
          onChange={event => setScale(event.target.value)} /></label>
        <label>{ADD_LIST_COPY.unit}<input className="gateway-number" value={unit} maxLength={16}
          onChange={event => setUnit(event.target.value)} /></label></>}
      <button type="button" onClick={check} disabled={!kind}>{range ? ADD_LIST_COPY.planRange : ADD_LIST_COPY.check}</button>
    </div>
    <SaveProblem text={problem} />
    {plan && <ListPlanRows plan={plan} protocol={protocol} types={types} />}
    {plan && <div className="row add-actions">
      <button type="button" className="primary" disabled={!planReady(plan)} onClick={() => add(plan.points)}>
        <Plus size={16} />{ADD_LIST_COPY.add(plan.points.length || plan.rows.length)}</button>
      {problems.length > 0 && <span className="small add-blocked">⚠ {ADD_LIST_COPY.fixFirst}</span>}
    </div>}
  </div>;
}
