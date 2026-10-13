import { RotateCcw, Zap } from 'lucide-react';
import { useState } from 'react';
import type { Signal } from '../../api/types';
import { SignalValueEditor } from '../../components/SignalValueEditor';
import { pulse as sendPulse, resume, setValue, type Point } from './signalActions';

/** One way to set, pulse or resume a simulated value, used on every screen (UI-06). */
export function ValueControl({ point, signal, value, running, perform, pulse, generator, segmented }: {
  point: Point; signal: Signal; value: unknown; running: boolean;
  perform: (task: () => Promise<unknown>) => Promise<void>;
  pulse?: { active: boolean; durationMs: number }; generator?: boolean; segmented?: boolean;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const writable = signal.writable && signal.fc !== 'CF';
  const run = async (task: () => Promise<unknown>) => {
    setBusy(true); setError('');
    await perform(async () => {
      try { await task(); }
      catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); throw reason; }
    });
    setBusy(false);
  };
  if (!running || !writable) return <span className="mono" title={writable ? 'Press Run to change it' : 'Read only'}>
    {value === undefined ? '—' : String(value)}</span>;
  return <div className="stack" style={{ gap: 6 }}>
    <div className="row" style={{ flexWrap: 'wrap' }}>
      <SignalValueEditor signal={signal} value={value} disabled={busy} variant={segmented ? 'segmented' : undefined}
        update={next => run(() => setValue(point, next))} />
      {pulse && signal.kind === 'BOOLEAN' && <button disabled={busy} title={`${pulse.durationMs} ms, then back`}
        onClick={() => run(() => sendPulse(point, pulse.active, pulse.durationMs))}>
        <Zap size={15} />Pulse {pulse.active ? 'True' : 'False'}</button>}
      {generator && <button className="ghost" disabled={busy} title="Let the generator drive this value again"
        onClick={() => run(() => resume(point))}><RotateCcw size={15} />Resume generator</button>}
    </div>
    {error && <span className="small" role="alert" style={{ color: 'var(--error)' }}>{error}</span>}
  </div>;
}
