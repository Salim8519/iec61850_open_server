import type { ControlModel } from '../api/types.ts';

/** Select before operate (ADR-0108), as the browser shows it. The Go engine holds the selection
 *  and refuses a late Operate; this state machine only says what to show and which button works.
 *
 *  idle --select--> selecting --selected--> selected --operate--> operating --done--> idle
 *                        |                      |  \--cancel--> cancelling --done--> idle
 *                        \--failed--> idle      \--tick past the deadline--> idle (expired)
 *  A Cancel the device refuses goes back to selected with its words: the device and the engine
 *  still hold the selection (change 0176). */
export type SboPhase = 'idle' | 'selecting' | 'selected' | 'operating' | 'cancelling';
/** What the last step said. The words always carry it; tone only picks the icon. */
export type SboTone = 'none' | 'done' | 'failed' | 'expired' | 'cancelled';
export type SboState<C = unknown> = {
  phase: SboPhase; choice: C | null; deadline: number | null; windowMs: number; tone: SboTone; message: string;
};
export type SboEvent<C = unknown> =
  | { type: 'select'; choice: C }
  | { type: 'selected'; windowMs: number; now: number }
  | { type: 'failed'; message: string }
  | { type: 'tick'; now: number }
  | { type: 'operate' }
  | { type: 'operated'; message: string }
  | { type: 'cancel' }
  | { type: 'cancelled'; message: string }
  | { type: 'cancelRefused'; message: string }
  | { type: 'clear' };

export const SBO_IDLE: SboState<never> = { phase: 'idle', choice: null, deadline: null, windowMs: 0, tone: 'none', message: '' };

const MODELS: readonly ControlModel[] = ['status-only', 'direct-normal', 'sbo-normal', 'direct-enhanced', 'sbo-enhanced'];

/** The model as text, from the numeric ctlModel of `controls` and `extended_controls`, or a model name. */
export function modelName(model: number | string | null | undefined): ControlModel | '' {
  if (typeof model === 'string') return (MODELS as readonly string[]).includes(model) ? model as ControlModel : '';
  return typeof model === 'number' && Number.isInteger(model) && model >= 0 && model < MODELS.length ? MODELS[model] : '';
}

/** True when a command to this model needs Select first. */
export const isSbo = (model: number | string | null | undefined): boolean => modelName(model).startsWith('sbo-');

/** Whole seconds left before the selection runs out (0 when it has). */
export function secondsLeft(state: Pick<SboState, 'deadline'>, now: number): number {
  return state.deadline === null ? 0 : Math.max(0, Math.ceil((state.deadline - now) / 1000));
}

/** One step of the flow. Events that do not fit the phase are ignored, so a late answer never
 *  revives a selection that was cancelled or ran out. */
export function sboReducer<C>(state: SboState<C>, event: SboEvent<C>): SboState<C> {
  switch (event.type) {
    case 'select':
      return state.phase === 'idle' ? { ...SBO_IDLE, phase: 'selecting', choice: event.choice } : state;
    case 'selected':
      if (state.phase !== 'selecting') return state;
      if (event.windowMs <= 0) return { ...state, phase: 'idle', deadline: null, tone: 'expired', message: '' };
      return { ...state, phase: 'selected', windowMs: event.windowMs, deadline: event.now + event.windowMs, tone: 'none', message: '' };
    case 'failed':
      return state.phase === 'idle' ? state : { ...state, phase: 'idle', deadline: null, tone: 'failed', message: event.message };
    case 'tick':
      return state.phase === 'selected' && state.deadline !== null && event.now >= state.deadline
        ? { ...state, phase: 'idle', deadline: null, tone: 'expired', message: '' } : state;
    case 'operate':
      return state.phase === 'selected' ? { ...state, phase: 'operating' } : state;
    case 'operated':
      return state.phase === 'operating' ? { ...state, phase: 'idle', deadline: null, tone: 'done', message: event.message } : state;
    case 'cancel':
      return state.phase === 'selected' ? { ...state, phase: 'cancelling' } : state;
    case 'cancelled':
      return state.phase === 'cancelling' ? { ...state, phase: 'idle', deadline: null, tone: 'cancelled', message: event.message } : state;
    case 'cancelRefused':
      return state.phase === 'cancelling' ? { ...state, phase: 'selected', tone: 'failed', message: event.message } : state;
    case 'clear':
      return state.phase === 'idle' ? { ...state, tone: 'none', message: '' } : state;
  }
  return state;
}

/** True while the flow holds a selection or waits for an answer (other commands wait). */
export const sboBusy = (state: Pick<SboState, 'phase'>): boolean => state.phase !== 'idle';
