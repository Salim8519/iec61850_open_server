import { useState } from 'react';
import { Send } from 'lucide-react';
import { sendGoose } from '../../../api/gooseMessages';
import type { EquipmentData, EquipmentNode, GooseLinkSetting, Project } from '../../../api/types';
import { GOOSE_LINK_COPY } from '../../../copy/iedComponent';
import { ACTION_LABEL, leafName, leafOf, linkActions, takesGooseLink, valueWords } from '../../../equipment/gooseLinkChoices';
import { GooseLinkPicker } from './GooseLinkPicker';
import { useIedMessages } from './useIedMessages';
import './ied-sheet.css';

type Perform = (task: () => Promise<unknown>) => Promise<void>;

/** The optional GOOSE section of an equipment sheet (ADR-0112). Stopped: add, change or remove
 * the link. Running: what it sends and the Send buttons; Go plans the values and writes them in
 * one batch, so the message's state number rises once. */
export function GooseLinkSection({ node, project, locked, perform, update }: {
  node: EquipmentNode; project: Project; locked: boolean; perform: Perform; update: (data: EquipmentData) => void;
}) {
  const data = node.data;
  const link = data.goose_link ?? null;
  const [adding, setAdding] = useState(false);
  if (!takesGooseLink(data.kind) || (locked && !link)) return null;
  return <section className="sheet-section"><h3>{GOOSE_LINK_COPY.title}</h3>
    {!link && !adding && <>
      <p className="small muted" style={{ margin: 0 }}>{GOOSE_LINK_COPY.hint}</p>
      <div className="sheet-actions"><button onClick={() => setAdding(true)}>{GOOSE_LINK_COPY.add}</button></div></>}
    {!locked && (link || adding) && <GooseLinkPicker key={node.id} kind={data.kind} link={link} ieds={project.ieds}
      save={next => update({ ...data, goose_link: next })}
      remove={() => { setAdding(false); if (link) update({ ...data, goose_link: null }); }} />}
    {!locked && link && <p className="small muted" style={{ margin: 0 }}>{GOOSE_LINK_COPY.stoppedSend}</p>}
    {locked && link && <GooseLinkSend nodeId={node.id} data={data} link={link} project={project} perform={perform} />}
  </section>;
}

function GooseLinkSend({ nodeId, data, link, project, perform }: {
  nodeId: string; data: EquipmentData; link: GooseLinkSetting; project: Project; perform: Perform;
}) {
  const { answer, refresh } = useIedMessages(link.ied_id, true);
  const [number, setNumber] = useState('');
  const [note, setNote] = useState('');
  const message = answer?.messages.find(item => item.gocb_ref === link.gocb_ref);
  const leaf = leafOf(answer?.messages, link.gocb_ref, link.one);
  const actions = linkActions(data.kind, link, leaf);
  const iedName = project.ieds.find(ied => ied.id === link.ied_id)?.name ?? link.ied_id;
  const why = answer && !answer.can_send ? answer.why_not : '';
  const send = (body: Parameters<typeof sendGoose>[1], words: string) => perform(async () => {
    setNote('');
    await sendGoose(nodeId, body);
    setNote(GOOSE_LINK_COPY.sent(words)); refresh();
  });
  const now = leaf ? valueWords(leaf) : '';
  return <>
    <p className="small goose-link-summary">{GOOSE_LINK_COPY.summary(iedName, message?.name ?? link.gocb_ref)}
      {leaf && <> · {leafName(leaf.ref)}{now && ` (now ${now})`}</>}</p>
    {why && <p className="small">{why}</p>}
    <div className="sheet-actions">
      {Array.isArray(actions) && actions.map(action => <button key={action} disabled={!!why || !answer} title={why}
        onClick={() => send({ action }, `${data.label} ${ACTION_LABEL[action]}`)}><Send size={14} />
        {GOOSE_LINK_COPY.send(ACTION_LABEL[action])}</button>)}
      {actions === 'number' && <>
        <input value={number} inputMode="decimal" aria-label={GOOSE_LINK_COPY.numberLabel} placeholder="Number" disabled={!!why}
          onChange={event => setNumber(event.target.value)} style={{ width: 110 }} />
        <button disabled={!!why || !number.trim() || !Number.isFinite(Number(number))} title={why}
          onClick={() => send({ action: 'value', value: Number(number) }, `${data.label} ${number.trim()}`)}>
          <Send size={14} />{GOOSE_LINK_COPY.send('value')}</button></>}
    </div>
    {note && <p className="small" role="status" style={{ margin: 0 }}>{note}</p>}
    <p className="small muted" style={{ margin: 0 }}>{GOOSE_LINK_COPY.lockedEdit}</p>
  </>;
}
