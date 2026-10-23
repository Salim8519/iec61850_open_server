import type { GooseMessage } from '../../../api/gooseMessages';
import { IED_SHEET_COPY } from '../../../copy/iedComponent';
import { GooseValueSend } from './GooseValueSend';

/** The GOOSE messages one IED sends (ADR-0112): the name, who receives it, and every value in
 * order, with send controls for the values the drawing may change. Real IEDs are watch only. */
export function GooseMessageList({ messages, canSend, why, send }: {
  messages: GooseMessage[]; canSend: boolean; why: string;
  send: (gocbRef: string, ref: string, value: boolean | number, words: string) => void;
}) {
  if (!messages.length) return <p className="small muted">{IED_SHEET_COPY.sendsNone}</p>;
  return <div className="goose-messages">{messages.map(message => <div key={message.gocb_ref} className="goose-message">
    <div className="goose-message-head"><strong title={message.gocb_ref}>{message.name}</strong>
      <span className="small muted">{IED_SHEET_COPY.to(message.subscribers)}</span></div>
    {!message.complete && <p className="small muted">{IED_SHEET_COPY.incomplete}</p>}
    <ol className="goose-members">{message.members.map(member => {
      const sendable = member.refs.filter(leaf => leaf.sendable && leaf.value_type);
      return <li key={member.position}><span className="goose-member-text">{member.text}</span>
        {sendable.map(leaf => <GooseValueSend key={leaf.ref} leaf={leaf} named={member.refs.length > 1} disabled={!canSend} why={why}
          send={(value, words) => send(message.gocb_ref, leaf.ref, value, words)} />)}
        {!sendable.length && <small className="muted"> · {IED_SHEET_COPY.notSendable}</small>}</li>;
    })}</ol>
  </div>)}</div>;
}
