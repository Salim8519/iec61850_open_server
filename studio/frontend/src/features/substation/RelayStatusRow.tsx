import { ArrowDown, ArrowUp, RotateCcw, Trash2, Zap } from 'lucide-react';
import type { RelayStatus, Signal } from '../../api/types';
import { SignalValueEditor } from '../../components/SignalValueEditor';
import { RelayValueChip } from '../../equipment/RelayPanel';
import { useLiveValue } from '../../equipment/liveContext';
import { pulseTarget, relayRowLabel, shortRef } from '../../equipment/relayDisplay';

export const PULSE_MS = 1000;

type Props = {
  entry: RelayStatus; iedName?: string; index: number; count: number; locked: boolean;
  signal?: Signal; liveKey: string; fallback: unknown; busy: boolean; manual: boolean;
  rename: (label: string) => void; move: (direction: number) => void; remove: () => void;
  initiate: (path: string, extra?: Record<string, unknown>) => void;
};

/** One relay signal card: rename/reorder while stopped, value and initiation while running. */
export function RelayStatusRow({ entry, iedName, index, count, locked, signal, liveKey, fallback, busy, manual,
  rename, move, remove, initiate }: Props) {
  const live = useLiveValue(liveKey);
  const value = live === undefined ? fallback : live;
  const triggerable = !!signal && signal.writable && signal.fc !== 'CF';
  const boolean = signal?.kind === 'BOOLEAN';
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
      <SignalValueEditor signal={signal} value={value} disabled={busy} variant="segmented"
        update={next => initiate('/signals/update', { value: next })} />
      {boolean && <button className="primary" disabled={busy}
        aria-label={`Pulse ${entry.label} to ${target ? 'True' : 'False'}`}
        title={`Drive ${target ? 'True' : 'False'} for ${PULSE_MS} ms, then return`}
        onClick={() => initiate('/signals/pulse', { active_value: target, duration_ms: PULSE_MS })}>
        <Zap size={14} />Pulse {target ? 'True' : 'False'}</button>}
      {manual && <button className="ghost" disabled={busy} title="Release the manual value"
        onClick={() => initiate('/signals/release-override')}><RotateCcw size={14} />Resume automatic</button>}
    </div>}
  </div>;
}
