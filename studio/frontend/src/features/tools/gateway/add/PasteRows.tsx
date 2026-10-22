import { ClipboardPaste, Download, Plus, Upload } from 'lucide-react';
import { useRef, useState } from 'react';
import type { GatewayPoint, Protocol } from '../../../../api/gateway';
import { addApi, fileBase64, type Plan } from '../../../../api/gatewayAdd';
import { ADD_COPY } from '../../../../copy/gatewayAdd';
import { SaveProblem } from '../../../../ui/SaveStatus';
import { Spinner } from '../../../../ui/Spinner';
import { saveFile } from '../saveFile';
import { canAdd, hasRealDevice, problemLines } from './addPlan';
import { PlannedRows } from './PlannedRows';
import { RealWarning } from './RealWarning';

const COLUMN_NAMES: Record<string, string> = { name: 'name', type: 'type', ied: 'IED', point: 'point', address: 'address', scale: 'scale' };

/** "Paste from Excel": paste rows (with or without a title row) or upload the filled Excel template. The engine
 *  recognises the columns, checks every row and marks bad rows in red with the reason; nothing is added until
 *  every row is right. */
export function PasteRows({ protocol, rows, add, fail }: {
  protocol: Protocol; rows: GatewayPoint[]; add: (points: GatewayPoint[]) => void; fail: (message: string) => void;
}) {
  const [text, setText] = useState('');
  const [plan, setPlan] = useState<Plan | null>(null);
  const [busy, setBusy] = useState(false);
  const [problem, setProblem] = useState('');
  const [understood, setUnderstood] = useState(false);
  const file = useRef<HTMLInputElement>(null);
  const check = (ask: () => Promise<Plan>) => {
    setBusy(true); setProblem('');
    setUnderstood(false);
    ask().then(setPlan).catch(error => { setPlan(null); setProblem(error.message); }).finally(() => setBusy(false));
  };
  const read = (value = text) => value.trim() ? check(() => addApi.pasted(protocol, value, rows)) : setProblem(ADD_COPY.emptyPaste);
  const upload = (picked: File) => check(() => fileBase64(picked).then(data => addApi.uploaded(protocol, data, rows)));
  const problems = problemLines(plan);
  const real = hasRealDevice(plan?.points ?? []);
  const found = (plan?.columns ?? []).filter(Boolean).map(column => COLUMN_NAMES[column] ?? column).join(', ');
  return <div className="add-steps">
    <label className="paste-label">{ADD_COPY.pasteLabel}
      <textarea className="paste-box" rows={6} value={text} placeholder={ADD_COPY.pastePlaceholder} spellCheck={false}
        onChange={event => { setText(event.target.value); setPlan(null); }}
        onPaste={event => { const pasted = event.clipboardData.getData('text'); if (pasted) { event.preventDefault(); setText(pasted); read(pasted); } }} />
    </label>
    <p className="small muted">{ADD_COPY.pasteHint}</p>
    <div className="row add-actions">
      <button type="button" onClick={() => read()} disabled={busy}><ClipboardPaste size={16} />{ADD_COPY.readRows}</button>
      <span className="gateway-spacer" />
      <strong className="small">{ADD_COPY.template}</strong>
      <button type="button" className="ghost" title={ADD_COPY.templateHint} onClick={() => saveFile(addApi.templateUrl(protocol))}>
        <Download size={16} />{ADD_COPY.download}</button>
      <button type="button" className="ghost" disabled={busy} onClick={() => file.current?.click()}><Upload size={16} />{ADD_COPY.upload}</button>
      <input ref={file} type="file" hidden accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        aria-label={ADD_COPY.upload} onChange={event => { const picked = event.target.files?.[0]; if (picked) upload(picked); event.target.value = ''; }} />
    </div>
    {busy && <Spinner inline label={ADD_COPY.reading} />}
    <SaveProblem text={problem} />
    {plan && found && <p className="small muted">{ADD_COPY.columnsFound(found)}</p>}
    {plan && <PlannedRows plan={plan} protocol={protocol} numbered />}
    {plan && real && <RealWarning understood={understood} change={setUnderstood} />}
    {plan && <div className="row add-actions">
      <button type="button" className="primary" disabled={!canAdd(plan, real, understood)} onClick={() => add(plan.points)}>
        <Plus size={16} />{ADD_COPY.add(plan.points.length || plan.rows.length)}</button>
      {problems.length > 0 && <span className="small add-blocked">⚠ {ADD_COPY.fixFirst}</span>}
    </div>}
  </div>;
}
