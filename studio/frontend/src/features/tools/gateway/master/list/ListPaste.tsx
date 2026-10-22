import { ClipboardPaste, Download, FileUp, Plus, Upload } from 'lucide-react';
import { useRef, useState, type RefObject } from 'react';
import { fileBase64 } from '../../../../../api/gatewayAdd';
import type { MasterProtocol } from '../../../../../api/master';
import { masterListApi, type ListPlan, type ListPoint, type MasterType } from '../../../../../api/masterList';
import { ADD_LIST_COPY } from '../../../../../copy/masterList';
import { SaveProblem } from '../../../../../ui/SaveStatus';
import { Spinner } from '../../../../../ui/Spinner';
import { saveFile } from '../../saveFile';
import { planProblems, planReady } from './listRows';
import { ListPlanRows } from './ListPlanRows';

const COLUMN_NAMES: Record<string, string> = { name: 'name', address: 'address', type: 'type', scale: 'scale',
  offset: 'offset', unit: 'unit', select: 'SBO', value: 'value', word_order: 'word order' };

/** "Paste from Excel": paste rows, upload the engine-made Excel template or a CSV export. The engine
 *  recognises the columns, checks every row and marks bad rows with ⚠ and the reason; nothing is added
 *  until every row is right. */
export function ListPaste({ protocol, types, add }: {
  protocol: MasterProtocol; types: MasterType[]; add: (points: ListPoint[]) => void;
}) {
  const [text, setText] = useState('');
  const [plan, setPlan] = useState<ListPlan | null>(null);
  const [busy, setBusy] = useState(false);
  const [problem, setProblem] = useState('');
  const workbook = useRef<HTMLInputElement>(null);
  const csv = useRef<HTMLInputElement>(null);
  const check = (ask: () => Promise<ListPlan>) => {
    setBusy(true); setProblem('');
    ask().then(setPlan).catch(error => { setPlan(null); setProblem(error.message); }).finally(() => setBusy(false));
  };
  const read = (value = text) => value.trim() ? check(() => masterListApi.rows(value)) : setProblem(ADD_LIST_COPY.emptyPaste);
  const problems = planProblems(plan);
  const found = (plan?.columns ?? []).filter(Boolean).map(column => COLUMN_NAMES[column] ?? column).join(', ');
  const picker = (ref: RefObject<HTMLInputElement | null>, accept: string, label: string, use: (file: File) => void) =>
    <input ref={ref} type="file" hidden accept={accept} aria-label={label}
      onChange={event => { const picked = event.target.files?.[0]; if (picked) use(picked); event.target.value = ''; }} />;
  return <div className="add-steps">
    <label className="paste-label">{ADD_LIST_COPY.pasteLabel}
      <textarea className="paste-box" rows={6} value={text} placeholder={ADD_LIST_COPY.pastePlaceholder} spellCheck={false}
        onChange={event => { setText(event.target.value); setPlan(null); }}
        onPaste={event => { const pasted = event.clipboardData.getData('text'); if (pasted) { event.preventDefault(); setText(pasted); read(pasted); } }} />
    </label>
    <p className="small muted">{ADD_LIST_COPY.pasteHint}</p>
    <div className="row add-actions">
      <button type="button" onClick={() => read()} disabled={busy}><ClipboardPaste size={16} />{ADD_LIST_COPY.readRows}</button>
      <span className="gateway-spacer" />
      <button type="button" className="ghost" title={ADD_LIST_COPY.templateHint} onClick={() => saveFile(masterListApi.templateUrl())}>
        <Download size={16} />{ADD_LIST_COPY.download}</button>
      <button type="button" className="ghost" disabled={busy} onClick={() => workbook.current?.click()}>
        <Upload size={16} />{ADD_LIST_COPY.upload}</button>
      <button type="button" className="ghost" disabled={busy} onClick={() => csv.current?.click()}>
        <FileUp size={16} />{ADD_LIST_COPY.uploadCsv}</button>
      {picker(workbook, '.xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', ADD_LIST_COPY.upload,
        picked => check(() => fileBase64(picked).then(masterListApi.uploaded)))}
      {picker(csv, '.csv,text/csv', ADD_LIST_COPY.uploadCsv, picked => check(() => picked.text().then(masterListApi.rows)))}
    </div>
    {busy && <Spinner inline label={ADD_LIST_COPY.reading} />}
    <SaveProblem text={problem} />
    {plan && found && <p className="small muted">{ADD_LIST_COPY.columnsFound(found)}</p>}
    {plan && <ListPlanRows plan={plan} protocol={protocol} types={types} numbered />}
    {plan && <div className="row add-actions">
      <button type="button" className="primary" disabled={!planReady(plan)} onClick={() => add(plan.points)}>
        <Plus size={16} />{ADD_LIST_COPY.add(plan.points.length || plan.rows.length)}</button>
      {problems.length > 0 && <span className="small add-blocked">⚠ {ADD_LIST_COPY.fixFirst}</span>}
    </div>}
  </div>;
}
