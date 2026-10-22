import { Check, Clock, TriangleAlert } from 'lucide-react';
import type { SaveState } from '../state/autoSave';
import { SpinIcon } from './Spinner';
import './ui.css';

/** The quiet "Saving… / Saved" words beside a section title (save always). Words carry the meaning;
 *  the icon only repeats them. A failed save says so here; the reason is the red line under the field. */
export function SaveStatus({ state, waiting = '' }: { state: SaveState; waiting?: string }) {
  const phase = waiting ? 'waiting' : state.phase;
  return <span className={`save-status ${phase}`} role="status" aria-live="polite">
    {phase === 'saving' && <><SpinIcon size={14} />Saving…</>}
    {phase === 'saved' && <><Check size={14} aria-hidden />Saved</>}
    {phase === 'failed' && <><TriangleAlert size={14} aria-hidden />Not saved yet</>}
    {phase === 'waiting' && <><Clock size={14} aria-hidden />{waiting || 'Will save in a moment'}</>}
  </span>;
}

/** One red line with words under the field or section whose value is not saved (role="alert"). */
export function SaveProblem({ text }: { text: string }) {
  return text ? <p className="small save-problem" role="alert">⚠ {text}</p> : null;
}
