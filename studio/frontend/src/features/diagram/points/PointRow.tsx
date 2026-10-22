import { useState } from 'react';
import { Send, SlidersHorizontal, TriangleAlert } from 'lucide-react';
import type { ExtendedControl, Ied, PanelPoint } from '../../../api/types';
import { useLiveValue } from '../../../equipment/liveContext';
import { isHealthy, kindOfPoint, showPoint } from '../../../equipment/pointDisplay';
import { pointRef, type PointPatch } from '../../../equipment/pointPanel';
import { controlOf, type CommandChoice, type CommandControl, type CommandValue, type PointAvailability }
  from '../../../equipment/pointRequests';
import { relayValueKey } from '../../../equipment/stateBindings';
import { shortRef } from '../../../equipment/relayDisplay';
import { EditableText } from '../../../ui/EditableText';
import { KindBadge, kindRow } from '../../../ui/KindBadge';
import { POINT_COPY } from '../../../copy/pointPanel';
import { PointRowEdit } from './PointRowEdit';
import { PointRowRun } from './PointRowRun';
import type { Outcome } from './usePointActions';

export type PointRowProps = {
  nodeId: string; point: PanelPoint; ied?: Ied; extended: readonly ExtendedControl[]; locked: boolean; available: PointAvailability; why: string;
  outcome?: Outcome; stopped: boolean;
  set: (point: PanelPoint, value: boolean | number, done: string) => void;
  send: (point: PanelPoint, value: CommandValue, done: string) => Promise<void>;
  change: (id: string, patch: PointPatch) => void; remove: (id: string) => void;
};

/** One point of the popup: kind badge, name, live value, and either the actions (running) or, while
 *  stopped, a settings button that opens the edit fields (one compact line until then). A command asks
 *  one small question before it is sent. */
export function PointRow({ nodeId, point, ied, extended, locked, available, why, outcome, stopped, set, send, change, remove }: PointRowProps) {
  const raw = useLiveValue(point.type === 'command' ? '' : relayValueKey(nodeId, point.id));
  const control: CommandControl | undefined = controlOf(point, ied, extended);
  const kind = kindOfPoint(point, control);
  const [pending, setPending] = useState<CommandChoice | null>(null);
  const [problem, setProblem] = useState('');
  const [editing, setEditing] = useState(false);
  const shown = point.type === 'command' ? null : showPoint(point, raw, stopped);
  const unhealthy = locked && point.type !== 'command' && point.healthy !== null && !stopped && !isHealthy(point, raw);
  const name = point.label === pointRef(point) ? shortRef(point.label, ied?.name) : point.label;
  return <div className="pp-row kind-edge" {...kindRow(kind)} title={pointRef(point)}>
    <div className="pp-row-line">
      <KindBadge kind={kind} raw={control?.cdc} />
      <span className="pp-name">
        {locked ? name : <EditableText value={name} label={`Name of ${point.label}`} placeholder="Name this point"
          onSave={text => change(point.id, { label: text.trim().slice(0, 100) || pointRef(point) })} />}
        {unhealthy && <small className="pp-flag"><TriangleAlert size={11} aria-hidden="true" /> {POINT_COPY.notHealthyOne}</small>}
      </span>
      {shown && <span className={`point-chip ${shown.state}`}>{shown.text}</span>}
      {locked && <PointRowRun nodeId={nodeId} point={point} control={control} raw={raw} disabled={!available.ok} why={why}
        set={(value, done) => set(point, value, done)} ask={choice => setPending(choice)} send={send} problem={setProblem} />}
      {!locked && <button className="icon ghost pp-settings" aria-expanded={editing}
        title={editing ? POINT_COPY.hideSettings : POINT_COPY.showSettings}
        aria-label={`${editing ? POINT_COPY.hideSettings : POINT_COPY.showSettings}: ${point.label}`}
        onClick={() => setEditing(open => !open)}><SlidersHorizontal size={15} /></button>}
    </div>
    {!locked && editing && <PointRowEdit point={point} change={patch => change(point.id, patch)} remove={() => remove(point.id)} />}
    {pending && <div className="confirm-box pp-confirm" role="alertdialog" aria-label="Confirm command">
      <span><strong>{POINT_COPY.confirmTitle(pending.label, point.label)}</strong> <small>{POINT_COPY.confirmHint}</small></span>
      <span className="pp-confirm-buttons">
        <button autoFocus onClick={() => setPending(null)}>{POINT_COPY.cancel}</button>
        <button className="danger" onClick={() => { const choice = pending; setPending(null); send(point, choice.value, choice.label); }}>
          <Send size={14} />{POINT_COPY.confirmSend}</button></span>
    </div>}
    {problem && <p className="relay-error" role="alert">{problem}</p>}
    {outcome && <p className={`pp-outcome ${outcome.state}`} role={outcome.state === 'failed' ? 'alert' : 'status'}>{outcome.text}</p>}
  </div>;
}
