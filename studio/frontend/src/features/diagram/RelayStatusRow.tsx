import { ArrowDown, ArrowUp, Trash2, Zap } from 'lucide-react';
import type { RelayStatus, Signal } from '../../api/types';
import { kindOfSignal } from '../../components/signalKind';
import { ValueControl } from '../test/ValueControl';
import { RelayValueChip } from '../../equipment/RelayPanel';
import { useLiveValue } from '../../equipment/liveContext';
import { useDataClasses } from '../../points/dataClasses';
import { useSignalValue } from '../../state/useLive';
import { EditableText } from '../../ui/EditableText';
import { KindBadge, kindRow } from '../../ui/KindBadge';
import { pulseTarget, relayRowLabel, shortRef } from '../../equipment/relayDisplay';

export const PULSE_MS = 1000;

type Props = {
  entry: RelayStatus; iedName?: string; index: number; count: number; locked: boolean;
  signal?: Signal; liveKey: string; fallbackKey: string; generator: boolean;
  rename: (label: string) => void; move: (direction: number) => void; remove: () => void;
  perform: (task: () => Promise<unknown>) => Promise<void>;
};

/** One relay signal card: rename/reorder while stopped, value and initiation while running. */
export function RelayStatusRow({ entry, iedName, index, count, locked, signal, liveKey, fallbackKey, generator,
  rename, move, remove, perform }: Props) {
  const live = useLiveValue(liveKey);
  const fallback = useSignalValue(fallbackKey);
  const value = live === undefined ? fallback : live;
  const triggerable = !!signal && signal.writable && signal.fc !== 'CF';
  const target = pulseTarget(value);
  const classOf = useDataClasses(entry.source.ied_id);
  const kind = signal ? kindOfSignal(signal, classOf) : undefined;
  return <div className={`relay-card ${locked ? 'live' : ''}${kind ? ' kind-edge' : ''}`} {...(kind ? kindRow(kind) : {})}>
    <div className="relay-card-head">
      <span className="relay-card-index" aria-hidden="true">{index + 1}</span>
      {locked ? <strong className="relay-card-name" title={entry.label}>{relayRowLabel(entry, iedName)}</strong> :
        <span className="relay-card-name"><EditableText value={relayRowLabel(entry, iedName)}
          label={`Signal ${index + 1} name`} placeholder="Name this signal"
          onSave={text => rename(text.trim().slice(0, 100) || entry.source.ref)} /></span>}
      {locked ? <RelayValueChip value={value} /> : <span className="relay-card-tools">
        <button className="icon ghost" title="Move up" aria-label={`Move ${entry.label} up`}
          disabled={index === 0} onClick={() => move(-1)}><ArrowUp size={14} /></button>
        <button className="icon ghost" title="Move down" aria-label={`Move ${entry.label} down`}
          disabled={index === count - 1} onClick={() => move(1)}><ArrowDown size={14} /></button>
        <button className="icon ghost relay-remove" title="Remove signal" aria-label={`Remove ${entry.label}`}
          onClick={remove}><Trash2 size={14} /></button></span>}
    </div>
    <div className="relay-card-meta">
      {entry.label !== entry.source.ref &&
        <code title={entry.source.ref}>{shortRef(entry.source.ref, iedName)}</code>}
      {signal && kind && <><KindBadge kind={kind} raw={signal.kind} /><span>{signal.fc}</span></>}
      {signal ? <span className={`relay-kind ${triggerable ? 'trigger' : 'display'}`}>
        {triggerable ? <><Zap size={11} />Triggerable</> : 'Display only'}</span> :
        <span className="muted">Loading…</span>}
    </div>
    {locked && signal && triggerable && <div className="relay-card-actions">
      <ValueControl point={entry.source} signal={signal} value={value} running={locked} perform={perform}
        segmented pulse={{ active: target, durationMs: PULSE_MS }} generator={generator} />
    </div>}
  </div>;
}
