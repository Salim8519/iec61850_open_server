import { useState } from 'react';
import type { EquipmentKind, GooseLinkSetting, Ied } from '../../../api/types';
import { GOOSE_LINK_COPY } from '../../../copy/iedComponent';
import { gooseSenders, isSwitch, leafName, linkChoices, linkComplete, storedLink, type LinkDraft }
  from '../../../equipment/gooseLinkChoices';
import { useIedMessages } from './useIedMessages';

const TYPE_WORDS = { onoff: 'on/off', position: 'position', number: 'number' } as const;

/** Choose what this equipment sends by GOOSE (ADR-0112): IED, then message, then value (or, for a
 * switch, open and closed as two on/off values). Only a whole choice is saved; Go checks it again
 * and refuses in plain words. Edited only while the IEDs are stopped. */
export function GooseLinkPicker({ kind, link, ieds, save, remove }: {
  kind: EquipmentKind; link: GooseLinkSetting | null; ieds: Ied[];
  save: (link: GooseLinkSetting) => void; remove: () => void;
}) {
  const [draft, setDraft] = useState<LinkDraft>(() => link ?? { mode: 'one' });
  const { answer, error } = useIedMessages(draft.ied_id, false);
  const senders = gooseSenders(ieds);
  const message = answer?.messages.find(item => item.gocb_ref === draft.gocb_ref);
  const choices = message ? linkChoices(message, kind, draft.mode) : [];
  const change = (patch: Partial<LinkDraft>) => {
    const next = { ...draft, ...patch };
    setDraft(next);
    if (linkComplete(next)) save(storedLink(next));
  };
  const valueSelect = (field: 'one' | 'open' | 'closed', label: string) => <label>{label}
    <select value={draft[field] ?? ''} onChange={event => change({ [field]: event.target.value || null })}>
      <option value="">{GOOSE_LINK_COPY.chooseValue}</option>
      {choices.map(leaf => <option key={leaf.ref} value={leaf.ref}>
        {`${leafName(leaf.ref)} (${leaf.value_type ? TYPE_WORDS[leaf.value_type] : leaf.kind})`}</option>)}
    </select></label>;
  if (!senders.length && !link) return <p className="small muted">{GOOSE_LINK_COPY.noPublishers}</p>;
  return <div className="goose-link-fields">
    <label>{GOOSE_LINK_COPY.ied}<select value={draft.ied_id ?? ''}
      onChange={event => change({ ied_id: event.target.value || undefined, gocb_ref: undefined, one: null, open: null, closed: null })}>
      <option value="">{GOOSE_LINK_COPY.chooseIed}</option>
      {senders.map(ied => <option key={ied.id} value={ied.id}>{ied.name}</option>)}
      {draft.ied_id && !senders.some(ied => ied.id === draft.ied_id) &&
        <option value={draft.ied_id}>{ieds.find(ied => ied.id === draft.ied_id)?.name ?? draft.ied_id}</option>}
    </select></label>
    {error && <p className="relay-error" role="alert">{error}</p>}
    {draft.ied_id && answer && <label>{GOOSE_LINK_COPY.message}<select value={draft.gocb_ref ?? ''}
      onChange={event => change({ gocb_ref: event.target.value || undefined, one: null, open: null, closed: null })}>
      <option value="">{GOOSE_LINK_COPY.chooseMessage}</option>
      {answer.messages.map(item => <option key={item.gocb_ref} value={item.gocb_ref}>{item.name}</option>)}
    </select></label>}
    {message && isSwitch(kind) && <label>{GOOSE_LINK_COPY.mode}<select value={draft.mode}
      onChange={event => change({ mode: event.target.value as LinkDraft['mode'], one: null, open: null, closed: null })}>
      <option value="one">{GOOSE_LINK_COPY.one}</option><option value="contacts">{GOOSE_LINK_COPY.contacts}</option>
    </select></label>}
    {message && (draft.mode === 'contacts'
      ? <>{valueSelect('open', GOOSE_LINK_COPY.openValue)}{valueSelect('closed', GOOSE_LINK_COPY.closedValue)}</>
      : valueSelect('one', isSwitch(kind) ? GOOSE_LINK_COPY.positionValue : GOOSE_LINK_COPY.value))}
    <div className="sheet-actions"><button className="ghost" onClick={remove}>{GOOSE_LINK_COPY.remove}</button></div>
  </div>;
}
