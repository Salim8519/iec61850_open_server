import { Check, Circle, Loader2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import './stepList.css';

export type StepState = 'waiting' | 'running' | 'done';
/** One step of a longer wait: its words, where it is, how long it took, and for the running step
 *  an optional count ("12 of 30 addresses") with a bar. */
export type Step = { key: string; label: string; state: StepState; ms?: number;
  done?: number; total?: number; unit?: string };

/** Shown after this long, so a wait never looks frozen. */
const CALM_MS = 4000;

const seconds = (ms: number) => ms < 1000 ? `${Math.max(ms, 0)} ms` : `${(ms / 1000).toFixed(1)} s`;

/** The one long-wait display (change 0150): every step with a tick when done, the running step
 *  turning with its count, waiting steps as empty circles, the time so far, and a calm line that
 *  nothing is stuck. Words and icons say each state; colour only repeats them. */
export function StepList({ title, steps }: { title: string; steps: Step[] }) {
  const [started] = useState(() => Date.now());
  const [now, setNow] = useState(started);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 250);
    return () => window.clearInterval(timer);
  }, []);
  const elapsed = now - started;
  return <div className="step-list" role="status" aria-live="polite">
    <div className="step-list-head"><strong>{title}</strong><span className="muted small">{seconds(elapsed)}</span></div>
    <ol>{steps.map(step => <li key={step.key} className={`step-${step.state}`}>
      <span className="step-icon" aria-hidden="true">{step.state === 'done' ? <Check size={16} />
        : step.state === 'running' ? <Loader2 size={16} className="ui-spin" /> : <Circle size={14} />}</span>
      <span className="sr-only">{step.state === 'done' ? 'Done: ' : step.state === 'running' ? 'Now: ' : 'Next: '}</span>
      <span className="step-text">
        <span>{step.label}</span>
        {step.state === 'running' && step.total ? <>
          <span className="small muted">{(step.done ?? 0).toLocaleString('en-US')} of {step.total.toLocaleString('en-US')} {step.unit ?? ''}</span>
          <span className="step-bar" aria-hidden="true"><span style={{ width: `${Math.min(100, 100 * (step.done ?? 0) / step.total)}%` }} /></span>
        </> : null}
      </span>
      {step.state === 'done' && step.ms !== undefined && <span className="small muted step-time">{seconds(step.ms)}</span>}
    </li>)}</ol>
    {elapsed > CALM_MS && <p className="small muted step-calm">Still working, nothing is stuck.</p>}
  </div>;
}
