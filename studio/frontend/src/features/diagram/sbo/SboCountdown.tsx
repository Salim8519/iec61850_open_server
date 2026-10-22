import { CircleCheck, CircleSlash, Send, Timer, TimerOff, TriangleAlert, X } from 'lucide-react';
import { SBO_COPY } from '../../../copy/selectOperate';
import { Spinner } from '../../../ui/Spinner';
import type { SelectOperate } from './useSelectOperate';
import './sbo.css';

type Props<C> = {
  flow: SelectOperate<C>;
  /** The picked command in words ("Close", "Tap 5", "Raise"). */
  what: (choice: C) => string;
  /** One line for a row of the point panel; the sheet uses the roomy layout. */
  compact?: boolean;
};

/** What a select-before-operate command is doing: selecting, the countdown with Operate and Cancel,
 *  or the last outcome. Every state has words and an icon, never colour alone (ADR-0108). */
export function SboCountdown<C>({ flow, what, compact }: Props<C>) {
  const { state, left } = flow;
  const picked = state.choice === null ? '' : what(state.choice);
  const layout = `sbo${compact ? ' compact' : ''}`;
  if (state.phase === 'selecting') {
    return <div className={layout} role="status"><Spinner label={SBO_COPY.selecting(picked)} /></div>;
  }
  if (state.phase !== 'idle') {
    const busy = state.phase !== 'selected';
    return <div className={`${layout} held`} role="group" aria-label={SBO_COPY.selected(picked, left)}>
      <span className="sbo-time" role="timer" aria-live="polite">
        <Timer size={15} aria-hidden="true" />
        <span>{compact ? SBO_COPY.selectedShort(left) : SBO_COPY.selected(picked, left)}</span>
      </span>
      <span className="sbo-buttons">
        <button type="button" className="danger" disabled={busy} title={SBO_COPY.operateHint} autoFocus
          onClick={() => { void flow.operate(); }}>
          <Send size={14} aria-hidden="true" />{state.phase === 'operating' ? SBO_COPY.operating : `${SBO_COPY.operate} ${picked}`}</button>
        <button type="button" disabled={busy} title={SBO_COPY.cancelHint} onClick={() => { void flow.cancel(); }}>
          <X size={14} aria-hidden="true" />{state.phase === 'cancelling' ? SBO_COPY.cancelling : SBO_COPY.cancel}</button>
      </span>
    </div>;
  }
  if (state.tone === 'none' || (state.tone === 'done' && !state.message)) return null;
  const outcome = state.tone === 'failed' ? { icon: <TriangleAlert size={14} aria-hidden="true" />, text: SBO_COPY.failed(state.message) }
    : state.tone === 'expired' ? { icon: <TimerOff size={14} aria-hidden="true" />, text: SBO_COPY.expired }
    : state.tone === 'cancelled' ? { icon: <CircleSlash size={14} aria-hidden="true" />, text: SBO_COPY.cancelled }
    : { icon: <CircleCheck size={14} aria-hidden="true" />, text: state.message };
  return <p className={`${layout} outcome ${state.tone}`} role={state.tone === 'failed' ? 'alert' : 'status'}>
    {outcome.icon}<span>{outcome.text}</span></p>;
}
