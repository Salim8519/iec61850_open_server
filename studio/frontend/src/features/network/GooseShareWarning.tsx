import { ShieldCheck, TriangleAlert } from 'lucide-react';
import type { GooseShare } from '../../api/network';
import { shareWarning } from '../../copy/gooseReach';
import './goose.css';

/** Confirm before simulated GOOSE leaves the PC; Cancel has the focus (GOOSE-19, GOOSE-25). */
export function GooseShareWarning({ share, busy, cancel, confirm }: {
  share: GooseShare; busy: boolean; cancel: () => void; confirm: () => void;
}) {
  const words = shareWarning(share.risk, share.uplink.label);
  const high = share.risk.level === 'high';
  return <div className={`goose-confirm${high ? ' real' : ''}`} role="alertdialog"
    aria-label={words.title}>
    <p><TriangleAlert size={18} aria-hidden /><strong>{words.title}</strong></p>
    <p className="small">{words.body}</p>
    {!!share.risk.reasons.length && <ul className="small">
      {share.risk.reasons.map(reason => <li key={reason}>{reason}</li>)}</ul>}
    <p className="small"><ShieldCheck size={16} aria-hidden />{words.guard}</p>
    <div className="row">
      <button onClick={cancel} autoFocus>{words.cancel}</button>
      <button className="danger" disabled={busy} onClick={confirm}>{words.confirm}</button>
    </div>
  </div>;
}
