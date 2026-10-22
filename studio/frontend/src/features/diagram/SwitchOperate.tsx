import { useState } from 'react';
import { Timer } from 'lucide-react';
import { api } from '../../api/client';
import type { CommandOptions, EquipmentNode, Project } from '../../api/types';
import { commandBody, optionsToSend, startOptions } from '../../equipment/commandOptions';
import { isSbo } from '../../equipment/selectOperate';
import { SBO_COPY } from '../../copy/selectOperate';
import { CommandOptionsPanel } from './CommandOptionsPanel';
import { SboCountdown } from './sbo/SboCountdown';
import { useSelectOperate } from './sbo/useSelectOperate';
import { useSboSteps } from './sbo/useSboSteps';

export type Operate = (nodeId: string, action: string, options?: CommandOptions) => Promise<void> | void;
type Action = 'open' | 'close';

/** Options changed in this tab but not sent yet, per node, so closing the sheet keeps them. */
const shown = new Map<string, CommandOptions>();
const route = (nodeId: string, step: string) => `/equipment/${encodeURIComponent(nodeId)}/${step}`;
const words = (action: Action) => action === 'open' ? 'Open' : 'Close';

/** Open and Close of a breaker, disconnector or earth switch (ADR-0108). A direct control sends at
 *  once, exactly as before. An SBO control does too by default (the engine selects and operates in
 *  one request, change 0173); with "step by step" on it selects first and shows the countdown with
 *  Operate and Cancel. The Go engine holds the selection; options are sent only once used. */
export function SwitchOperate({ node, project, busy, operate }: {
  node: EquipmentNode; project: Project; busy: boolean; operate: Operate;
}) {
  const binding = node.data.control_binding;
  const control = project.ieds.find(ied => ied.id === binding?.ied_id)?.controls.find(item => item.ref === binding?.ref);
  const sbo = isSbo(control?.model);
  const steps = useSboSteps() && sbo;
  const [options, setOptions] = useState(() => shown.get(node.id) ?? startOptions(node.data.command_options));
  const change = (next: CommandOptions) => { shown.set(node.id, next); setOptions(next); };
  const send = optionsToSend(node.data.command_options, options);
  const flow = useSelectOperate<Action>({
    select: action => api<{ expires_ms: number }>(route(node.id, 'select'), 'POST', commandBody(action, send)),
    operate: async action => { await operate(node.id, action, send); },
    cancel: () => api(route(node.id, 'cancel'), 'POST', {}),
  });
  const held = flow.state.phase !== 'idle';
  const press = (action: Action) => { if (steps) void flow.select(action); else void operate(node.id, action, send); };
  const hint = !sbo ? SBO_COPY.directHint : steps ? SBO_COPY.sboHint : SBO_COPY.sboOneClick;
  return <>
    <div className="sheet-actions">
      <button disabled={busy || held} title={hint} onClick={() => press('open')}>Open</button>
      <button className="primary" disabled={busy || held} title={hint} onClick={() => press('close')}>Close</button>
    </div>
    {sbo && <p className="small muted sbo-model"><Timer size={13} aria-hidden="true" />{hint}</p>}
    <SboCountdown flow={flow} what={words} />
    <CommandOptionsPanel options={options} change={change} disabled={held} />
  </>;
}
