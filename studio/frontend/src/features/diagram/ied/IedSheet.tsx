import { useState } from 'react';
import { sendGoose } from '../../../api/gooseMessages';
import type { EquipmentData, EquipmentNode, Project } from '../../../api/types';
import { GOOSE_LINK_COPY, IED_SHEET_COPY } from '../../../copy/iedComponent';
import { GooseMessageList } from './GooseMessageList';
import { useIedMessages } from './useIedMessages';
import './ied-sheet.css';

/** Sheet of an IED component (ADR-0112): which IED it is (one component per IED), the messages it
 * sends with their values, and the messages it receives. While the IEDs run, a simulated IED's
 * values can be changed here: Go writes them in one batch and the IED sends GOOSE. */
export function IedSheet({ node, project, locked, perform, update }: {
  node: EquipmentNode; project: Project; locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void>;
  update: (data: EquipmentData) => void;
}) {
  const data = node.data;
  const { answer, error, refresh } = useIedMessages(data.ied_id, locked);
  const [note, setNote] = useState('');
  const taken = new Set(project.diagram.nodes.filter(item => item.data.kind === 'IED' && item.id !== node.id)
    .map(item => item.data.ied_id));
  const choose = (iedId: string) => {
    const ied = project.ieds.find(item => item.id === iedId);
    update({ ...data, ied_id: ied?.id ?? null, label: ied?.name ?? data.label });
  };
  const send = (gocbRef: string, ref: string, value: boolean | number, words: string) => perform(async () => {
    setNote('');
    await sendGoose(node.id, { gocb_ref: gocbRef, ref, value });
    setNote(GOOSE_LINK_COPY.sent(words)); refresh();
  });
  return <>
    <section className="sheet-section"><h3>{IED_SHEET_COPY.pickTitle}</h3>
      <label>IED<select disabled={locked} value={data.ied_id ?? ''} onChange={event => choose(event.target.value)}>
        <option value="">{IED_SHEET_COPY.pickNone}</option>
        {project.ieds.map(ied => <option key={ied.id} value={ied.id} disabled={taken.has(ied.id)}>
          {taken.has(ied.id) ? IED_SHEET_COPY.taken(ied.name) : ied.name}</option>)}</select></label>
      <p className="small muted" style={{ margin: 0 }}>{locked ? IED_SHEET_COPY.lockedPick : IED_SHEET_COPY.pickHint}</p>
    </section>
    {data.ied_id && <section className="sheet-section"><h3>{IED_SHEET_COPY.sends}</h3>
      {error && <p className="relay-error" role="alert">{error}</p>}
      {!answer && !error && <p className="small muted">{IED_SHEET_COPY.loading}</p>}
      {answer && <>
        {answer.role === 'real' ? <p className="small">{IED_SHEET_COPY.watchOnly}</p>
          : answer.can_send ? <p className="small muted">{IED_SHEET_COPY.sendHint}</p>
          : answer.why_not && <p className="small">{answer.why_not}</p>}
        <GooseMessageList messages={answer.messages} canSend={answer.can_send} why={answer.why_not} send={send} />
        {note && <p className="small" role="status">{note}</p>}
      </>}
    </section>}
    {answer && <section className="sheet-section"><h3>{IED_SHEET_COPY.receives}</h3>
      {answer.receives.length ? <ul className="goose-receives">{answer.receives.map(item =>
        <li key={`${item.publisher}:${item.gocb_ref ?? item.block}`}><strong>{item.block}</strong> from {item.publisher}
          {item.status && <small className="muted"> · {IED_SHEET_COPY.receiveStatus(item.status)}</small>}</li>)}</ul>
        : <p className="small muted">{IED_SHEET_COPY.receivesNone}</p>}
    </section>}
  </>;
}
