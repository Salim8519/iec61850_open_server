import { type ReactNode } from 'react';
import { api } from '../../../api/client';
import type { PanelPoint } from '../../../api/types';
import { cancelBody, commandChoices, pointRoute, selectBody, type CommandChoice, type CommandControl, type CommandValue }
  from '../../../equipment/pointRequests';
import { isSbo } from '../../../equipment/selectOperate';
import { commandInput, commandWords } from '../../../equipment/typedCommand';
import { ListBox, NumberBox } from './CommandInputs';
import { SBO_COPY } from '../../../copy/selectOperate';
import { SboCountdown } from '../sbo/SboCountdown';
import { StepButtons } from '../sbo/StepButtons';
import { useSelectOperate } from '../sbo/useSelectOperate';
import { useSboSteps } from '../sbo/useSboSteps';

type Props = {
  nodeId: string; point: PanelPoint; control?: CommandControl; disabled: boolean; why: string;
  /** A direct command asks one small question first (the row shows it). */
  ask: (choice: CommandChoice) => void;
  send: (point: PanelPoint, value: CommandValue, done: string) => Promise<void>;
  problem: (text: string) => void;
};

/** What a command row offers while the IEDs run (ADR-0108). A command asks once and sends (an SBO
 *  control is selected and operated in one request, change 0173); with "step by step" on, an SBO
 *  control selects first and shows the countdown with Operate and Cancel below the row. */
export function PointCommand({ nodeId, point, control, disabled, why, ask, send, problem }: Props) {
  const steps = useSboSteps();
  const sbo = isSbo(control?.model) && steps;
  const words = (value: CommandValue) => commandWords(point, control, value);
  const flow = useSelectOperate<CommandValue>({
    select: value => api<{ expires_ms: number }>(pointRoute(nodeId), 'POST', selectBody(point, value)),
    operate: value => send(point, value, words(value)),
    cancel: () => api(pointRoute(nodeId), 'POST', cancelBody(point)),
  });
  const held = flow.state.phase !== 'idle';
  const off = disabled || held;
  const title = disabled ? why : sbo ? SBO_COPY.sboHint : isSbo(control?.model) ? SBO_COPY.sboOneClick : '';
  const verb = sbo ? 'Select' : 'Send';
  const pick = (value: CommandValue) => {
    flow.clear();
    if (sbo) void flow.select(value); else ask({ label: words(value), value });
  };
  const input = commandInput(control);
  let body: ReactNode;
  if (input === 'none') body = <span className="small muted">This IED has no such command any more.</span>;
  else if (input === 'number') body = <NumberBox point={point} control={control} disabled={off} title={title} verb={verb} pick={pick} problem={problem} />;
  else if (input === 'list') body = <ListBox point={point} control={control} disabled={off} title={title} verb={verb} pick={pick} />;
  else if (input === 'steps') body = <StepButtons disabled={off} title={title} label={`${verb} ${point.label}`} press={pick} />;
  else {
    body = <span className="pp-actions" role="group" aria-label={`Send ${point.label}`}>
      {commandChoices(control).map(choice => <button key={choice.label} type="button" className="pp-choice wire" disabled={off}
        title={title || `${choice.label} to this command`} onClick={() => pick(choice.value)}>
        {sbo ? choice.label.replace(/^Send /, 'Select ') : choice.label}</button>)}
    </span>;
  }
  return <>{body}<SboCountdown flow={flow} what={words} compact /></>;
}
