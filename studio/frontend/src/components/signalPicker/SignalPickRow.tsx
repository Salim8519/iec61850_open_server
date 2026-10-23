import type { Kind } from '../../points/pointKind';
import { KindBadge, kindRow } from '../../ui/KindBadge';
import { shortRef } from '../../equipment/relayDisplay';
import type { PickedSignal } from './pickerAnswer';

/** One point of a picker: the type badge and coloured edge, the description (or short reference), the
 *  reference, FC and whether it can be changed, then Choose; the chosen point says Chosen. */
export function SignalPickRow({ signal, kind, iedName, chosen, taken, replacing, disabled, chooseLabel, note, choose }: {
  signal: PickedSignal; kind: Kind; iedName?: string; chosen: boolean; taken: boolean; replacing: boolean;
  disabled: boolean; chooseLabel: string; note?: string; choose: () => void;
}) {
  const triggerable = signal.writable && signal.fc !== 'CF';
  const short = shortRef(signal.ref, iedName);
  return <div className={`signal-search-row kind-edge${chosen ? ' chosen' : ''}`} {...kindRow(kind)}>
    <KindBadge kind={kind} raw={signal.kind} />
    <span className="signal-search-text" title={signal.ref}>
      <strong>{signal.description || short}</strong>
      <small>{signal.description ? `${short} · ` : ''}{signal.fc}{' · '}{triggerable ? 'can be changed' : 'display only'}</small>
      {note && <small className="signal-search-note">{note}</small>}</span>
    {chosen ? <span className="badge live">Chosen ✓</span> :
      taken ? <span className="badge live">Added ✓</span> :
      <button className={triggerable && !replacing ? 'primary' : ''} disabled={disabled}
        title={replacing ? 'Use this point instead' : triggerable ? 'Choose this signal' : 'Choose as display-only'}
        onClick={choose}>{replacing ? 'Use this' : chooseLabel}</button>}
  </div>;
}
