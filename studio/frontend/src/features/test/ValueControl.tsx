import { RotateCcw, Zap } from 'lucide-react';
import { useState } from 'react';
import type { Signal } from '../../api/types';
import { SignalValueEditor } from '../../components/SignalValueEditor';
import { AutoValue } from '../../ui/EditableText';
import { pulse as sendPulse, resume, setValue, type Point } from './signalActions';

/** One way to set, pulse or resume a simulated value, used on every screen (UI-06). `generator`: a
 *  generator exists for this point; `driven`: it is driving the value right now (not overridden by hand),
 *  so the value shows the "auto" tag and there is nothing to resume yet. */
export function ValueControl({ point, signal, value, running, perform, pulse, generator, driven, segmented }: {
  point: Point; signal: Signal; value: unknown; running: boolean;
  perform: (task: () => Promise<unknown>) => Promise<void>;
  pulse?: { active: boolean; durationMs: number }; generator?: boolean; driven?: boolean; segmented?: boolean;
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
      {generator && driven && <AutoValue value="Generator" how="generator" />}
      <SignalValueEditor signal={signal} value={value} disabled={busy} variant={segmented ? 'segmented' : undefined}
        update={next => run(() => setValue(point, next))} />
      {pulse && signal.kind === 'BOOLEAN' && <button disabled={busy} title={`${pulse.durationMs} ms, then back`}
        onClick={() => run(() => sendPulse(point, pulse.active, pulse.durationMs))}>
        <Zap size={15} />Pulse {pulse.active ? 'True' : 'False'}</button>}
      {generator && !driven && <button className="ghost" disabled={busy} title="Let the generator drive this value again"
        onClick={() => run(() => resume(point))}><RotateCcw size={15} />Resume generator</button>}
    </div>
    {error && <span className="small" role="alert" style={{ color: 'var(--error)' }}>{error}</span>}
  </div>;
}
