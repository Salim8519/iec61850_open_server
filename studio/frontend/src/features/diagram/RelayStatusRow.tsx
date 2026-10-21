import { ArrowDown, ArrowUp, Trash2, Zap } from 'lucide-react';
import type { RelayStatus, Signal } from '../../api/types';
import { ValueControl } from '../test/ValueControl';
import { RelayValueChip } from '../../equipment/RelayPanel';
import { useLiveValue } from '../../equipment/liveContext';
import { useSignalValue } from '../../state/useLive';
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
  return <div className={`relay-card ${locked ? 'live' : ''}`}>
    <div className="relay-card-head">
      <span className="relay-card-index" aria-hidden="true">{index + 1}</span>
      {locked ? <strong className="relay-card-name" title={entry.label}>{relayRowLabel(entry, iedName)}</strong> :
        <input className="relay-card-name" maxLength={100} value={relayRowLabel(entry, iedName)}
          aria-label={`Signal ${index + 1} name`} placeholder="Name this signal" title="Rename"
          onChange={e => rename(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') (e.target as HTMLInputElement).blur(); }} />}
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
      {signal && <span>{signal.kind} · {signal.fc}</span>}
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
