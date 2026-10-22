import { Plus, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { gatewayApi, type GatewayPoint, type PointKind, type Protocol } from '../../../api/gateway';
import { ADDRESS_LABEL, KIND_WHERE, OWN_COPY, OWN_KIND_LABELS } from '../../../copy/gateway';
import { SaveProblem } from '../../../ui/SaveStatus';
import { SegmentedControl } from '../../../ui/SegmentedControl';
import { Spinner } from '../../../ui/Spinner';
import { isCommand, isMeasured, ownTargets, parseOwnNumber, statusChoices, statusNumber } from './ownPoints';

/** The small form behind "Add point": a made-up point with its own value. The backend suggests
 *  the next own id and the next free address. "Add to the list" adds the point and the list is saved at once,
 *  so SCADA gets it right away (there is no draft step any more). A wrong number keeps its text and shows
 *  one red line under the fields. */
export function AddOwnPoint({ protocol, rows, add, close, fail }: {
  protocol: Protocol; rows: GatewayPoint[]; add: (point: GatewayPoint) => void; close: () => void;
  fail: (message: string) => void;
}) {
  const [point, setPoint] = useState<GatewayPoint | null>(null);
  const [number, setNumber] = useState('0');
  const [problem, setProblem] = useState('');
  const suggest = (kind: PointKind, description = '') => gatewayApi.newOwnPoint(protocol, kind, rows)
    .then(answer => { setPoint({ ...answer.point, description }); setNumber(String(Number(answer.point.value ?? 0))); })
    .catch(error => { fail(error.message); close(); });
  useEffect(() => { suggest('single'); }, [protocol]);
  if (!point) return <div className="own-form"><Spinner label="Preparing a new point…" inline /></div>;
  const change = (patch: Partial<GatewayPoint>) => setPoint({ ...point, ...patch });
  const confirm = () => {
    let value = point.value;
    if (isMeasured(point.kind)) {
      const parsed = parseOwnNumber(number, point.kind);
      if ('error' in parsed) { setProblem(parsed.error); return; }
      value = parsed.value;
    }
    setProblem('');
    add({ ...point, value, description: point.description.trim(), ied_name: OWN_COPY.ownPoint, ied_role: 'own',
      monitor_only: false, draft: true });
  };
  const targets = ownTargets(rows);
  return <div className="own-form" role="group" aria-label={OWN_COPY.title}>
    <div className="row gateway-toolbar"><strong>{OWN_COPY.title}</strong><span className="gateway-spacer" />
      <button className="ghost icon-button" onClick={close} aria-label={OWN_COPY.cancel}><X size={16} /></button></div>
    <p className="small muted own-hint">{isCommand(point.kind) ? OWN_COPY.commandHint : OWN_COPY.hint}</p>
    <div className="row own-form-fields">
      <label>{OWN_COPY.name}<input value={point.description} placeholder={OWN_COPY.namePlaceholder} maxLength={120}
        onChange={event => change({ description: event.target.value })} /></label>
      <label>{OWN_COPY.type}<select value={point.kind} onChange={event => suggest(event.target.value as PointKind, point.description)}>
        {(Object.keys(OWN_KIND_LABELS) as PointKind[]).map(kind => <option key={kind} value={kind}>{OWN_KIND_LABELS[kind]}</option>)}
      </select></label>
      <label title={`${OWN_COPY.type}: ${KIND_WHERE[protocol][point.kind]}`}>{ADDRESS_LABEL[protocol]}
        <input className="gateway-number" inputMode="numeric" value={point.address}
          onChange={event => change({ address: Number(event.target.value.replace(/\D/g, '')) })} /></label>
      {isMeasured(point.kind) && <>
        <label>{OWN_COPY.value}<input className="gateway-number" inputMode="decimal" value={number}
          onChange={event => { setNumber(event.target.value); setProblem(''); }} /></label>
        <label title={OWN_COPY.scaleHint}>{OWN_COPY.scale}<input className="gateway-number" inputMode="decimal" value={point.scale}
          onChange={event => change({ scale: Number(event.target.value) || 0 })} /></label></>}
      {point.kind === 'single' && <div className="own-field"><span>{OWN_COPY.value}</span><SegmentedControl label={OWN_COPY.value}
        value={String(statusNumber(point.kind, point.value))} options={statusChoices(point.kind)}
        onChange={value => change({ value: Number(value) })} /></div>}
      {point.kind === 'double' && <label>{OWN_COPY.value}<select value={String(statusNumber(point.kind, point.value))}
        onChange={event => change({ value: Number(event.target.value) })}>
        {statusChoices(point.kind).map(choice => <option key={choice.value} value={choice.value}>{choice.label}</option>)}
      </select></label>}
      {isCommand(point.kind) && <label>{OWN_COPY.drives}<select value={point.target ?? ''}
        onChange={event => change({ target: event.target.value || undefined })}>
        <option value="">{OWN_COPY.drivesNone}</option>
        {targets.map(choice => <option key={choice.value} value={choice.value}>{choice.label}</option>)}
      </select></label>}
      <button className="primary" onClick={confirm}><Plus size={16} />{OWN_COPY.confirm}</button>
      <button className="ghost" onClick={close}>{OWN_COPY.cancel}</button>
    </div>
    <SaveProblem text={problem} />
  </div>;
}
