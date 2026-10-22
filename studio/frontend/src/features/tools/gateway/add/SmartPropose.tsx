import { Sparkles, X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { GatewayPoint, Protocol } from '../../../../api/gateway';
import { addApi, type IedSummary, type SmartAnswer } from '../../../../api/gatewayAdd';
import { ADD_COPY, iedBadge, SMART_COPY } from '../../../../copy/gatewayAdd';
import { SegmentedControl } from '../../../../ui/SegmentedControl';
import { Spinner } from '../../../../ui/Spinner';
import { StatusPill } from '../../../../ui/StatusPill';
import '../../../ieds/import-selection.css';
import './add-points.css';

type Mode = 'add_new' | 'replace';

/** Propose from IEDs in smart mode (ADR-0105): a preview with one tick per IED before anything changes.
 *  "Add only new points" keeps the list exactly as edited; "Replace the IED points" proposes them again and
 *  keeps own points. Names come from the SCL and each command gets its status point at the matching address. */
export function SmartPropose({ protocol, rows, apply, close, fail }: {
  protocol: Protocol; rows: GatewayPoint[]; apply: (answer: SmartAnswer) => void; close: () => void;
  fail: (message: string) => void;
}) {
  const [mode, setMode] = useState<Mode>(rows.length ? 'add_new' : 'replace');
  const [ticked, setTicked] = useState<string[] | null>(null); // null: every IED (the first answer lists them)
  const [answer, setAnswer] = useState<SmartAnswer | null>(null);
  const [busy, setBusy] = useState(true);
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => { dialog.current?.showModal(); }, []);
  useEffect(() => {
    let live = true;
    setBusy(true);
    addApi.smart(protocol, mode, rows, ticked ?? undefined).then(next => {
      if (!live) return;
      setAnswer(next);
      if (ticked === null && next.summary) setTicked(next.summary.filter(item => item.ticked).map(item => item.id));
    }).catch(error => { fail(error.message); close(); }).finally(() => { if (live) setBusy(false); });
    return () => { live = false; };
  }, [mode, ticked?.join(',')]);
  const summary: IedSummary[] = answer?.summary ?? [];
  const flip = (id: string) => setTicked(current => (current ?? []).includes(id)
    ? (current ?? []).filter(item => item !== id) : [...(current ?? []), id]);
  const none = ticked !== null && ticked.length === 0;
  const fromTicked = summary.filter(item => ticked?.includes(item.id)).reduce((sum, item) => sum + item.offered, 0);
  return <dialog ref={dialog} className="import-dialog add-points smart-propose" aria-labelledby="smart-title"
    onCancel={event => { event.preventDefault(); close(); }}>
      <div className="row add-points-head"><h3 id="smart-title">{SMART_COPY.title}</h3><span className="gateway-spacer" />
        <button type="button" className="ghost icon-button" onClick={close} aria-label={SMART_COPY.cancel} title={SMART_COPY.cancel}>
          <X size={18} /></button></div>
      <p className="small muted add-points-hint">{SMART_COPY.hint}</p>
      <SegmentedControl label={SMART_COPY.title} value={mode} onChange={setMode}
        options={(['add_new', 'replace'] as Mode[]).map(key => ({ value: key, label: SMART_COPY.modes[key], title: SMART_COPY.modeHints[key] }))} />
      <p className="small add-points-hint">{SMART_COPY.modeHints[mode]}</p>
      <div className="add-points-body">
        <h4>{SMART_COPY.ieds}</h4>
        {!answer && <Spinner label={SMART_COPY.loading} />}
        <div className="smart-ieds">{summary.map(item => <label key={item.id} className="row smart-ied">
          <input type="checkbox" checked={!!ticked?.includes(item.id)} onChange={() => flip(item.id)} />
          <strong>{item.name}</strong><StatusPill status={iedBadge(item.role)} />
          <span className="small muted">{SMART_COPY.offered(item.offered, item.new)}</span></label>)}</div>
        {answer && !none && <p className="small" role="status">{SMART_COPY.result(fromTicked, answer.points.length)}</p>}
        {none && <p className="small add-blocked">⚠ {SMART_COPY.noIeds}</p>}
      </div>
      <div className="row add-actions">
        <button type="button" className="primary" disabled={busy || none || !answer} onClick={() => answer && apply(answer)}>
          <Sparkles size={16} />{SMART_COPY.apply(answer?.points.length ?? 0)}</button>
        {busy && <Spinner inline size={14} label={ADD_COPY.check} />}
        <button type="button" className="ghost" onClick={close}>{SMART_COPY.cancel}</button>
      </div>
  </dialog>;
}
