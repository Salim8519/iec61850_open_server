import { ArrowLeftRight } from 'lucide-react';
import type { MasterProtocol, Role } from '../../../api/master';
import { DIAGRAM, ROLE_HINTS, ROLE_NAMES, SIDES } from '../../../copy/master';
import { HelpTip } from '../../../ui/HelpTip';
import { SegmentedControl } from '../../../ui/SegmentedControl';

const ROLES: Role[] = ['gateway', 'master', 'converter'];

/** "Act as" in the tool's header row: one role runs at a time, switching is locked while something
 *  runs, and "which side am I?" is a one-line caption instead of a diagram (change 0093). */
export function RoleSwitch({ role, locked, change, protocol, upstream }: {
  role: Role; locked: boolean; change: (role: Role) => void; protocol: MasterProtocol; upstream: MasterProtocol;
}) {
  return <div className="role-switch">
    <strong>Act as</strong>
    <SegmentedControl label="Act as" value={role} disabled={locked} onChange={change}
      options={ROLES.map(value => ({ value, label: ROLE_NAMES[value], title: ROLE_HINTS[value] }))} />
    <HelpTip text={`${ROLE_HINTS[role]}${locked ? ' Stop or disconnect to switch roles.' : ''}`} label="What does each role do?" />
    <SideCaption role={role} protocol={protocol} upstream={upstream} />
  </div>;
}

/** SCADA ⇄ ScadaBench ⇄ IEDs (or device) in one line; ScadaBench's place is marked in words. */
export function SideCaption({ role, protocol, upstream }: { role: Role; protocol: MasterProtocol; upstream: MasterProtocol }) {
  const words = SIDES[role === 'gateway' ? upstream : protocol];
  const up = SIDES[upstream];
  const steps = role === 'master'
    ? [{ name: 'ScadaBench', word: words.usShort, here: true }, { name: 'Gateway · RTU', word: words.deviceShort, here: false }]
    : role === 'converter'
    ? [{ name: 'SCADA', word: up.usShort, here: false },
      { name: 'ScadaBench', word: `${up.deviceShort} ⇄ ${words.usShort}`, here: true },
      { name: 'Gateway · RTU', word: words.deviceShort, here: false }]
    : [{ name: 'SCADA', word: words.usShort, here: false },
      { name: 'ScadaBench', word: `${words.deviceShort} ⇄ IEC 61850 client`, here: true },
      { name: 'IEDs', word: 'IEC 61850', here: false }];
  return <span className="side-caption" aria-label={DIAGRAM.title}>
    {steps.map((step, index) => <span key={step.name} className="side-caption-step">
      {index > 0 && <ArrowLeftRight size={14} aria-hidden />}
      <span className={step.here ? 'side-here-box' : ''} title={step.word}>
        <strong>{step.name}</strong>{step.here && <span className="sr-only"> (you are here)</span>}
        <span className="small muted"> {step.word}</span></span>
    </span>)}
  </span>;
}
