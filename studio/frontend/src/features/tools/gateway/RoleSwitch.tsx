import { ArrowLeftRight } from 'lucide-react';
import type { MasterProtocol, Role } from '../../../api/master';
import { DIAGRAM, ROLE_HINTS, ROLE_NAMES, SIDES } from '../../../copy/master';
import { InfoTip } from '../../../ui/InfoTip';
import { SegmentedControl } from '../../../ui/SegmentedControl';

const ROLES: Role[] = ['gateway', 'master', 'converter'];
const TERMS: Record<Role, string> = { gateway: 'station_gateway', master: 'controlling_station', converter: 'protocol_converter' };

/** "Act as": one role runs at a time; switching is locked while something runs and the diagram
 * folds away so the points get the screen. */
export function RoleSwitch({ role, locked, change, protocol, upstream }: {
  role: Role; locked: boolean; change: (role: Role) => void; protocol: MasterProtocol; upstream: MasterProtocol;
}) {
  return <div className="card role-switch">
    <div className="row role-row">
      <strong>Act as</strong>
      <SegmentedControl label="Act as" value={role} disabled={locked} onChange={change}
        options={ROLES.map(value => ({ value, label: ROLE_NAMES[value], title: ROLE_HINTS[value] }))} />
      <span className="small muted">{locked ? 'Stop or disconnect to switch roles.'
        : <InfoTip term={TERMS[role]} label={ROLE_HINTS[role]} />}</span>
    </div>
    {!locked && <SideDiagram role={role} protocol={protocol} upstream={upstream} />}
  </div>;
}

/** SCADA ⇄ gateway/RTU ⇄ IEDs with ScadaBench's place highlighted, and each side's word. */
function SideDiagram({ role, protocol, upstream }: { role: Role; protocol: MasterProtocol; upstream: MasterProtocol }) {
  const words = SIDES[role === 'gateway' ? upstream : protocol];
  const up = SIDES[upstream];
  const boxes = role === 'master'
    ? [{ name: DIAGRAM.scada, word: words.us, here: true }, { name: DIAGRAM.device, word: words.device, here: false }]
    : role === 'converter'
    ? [{ name: DIAGRAM.scada, word: up.us, here: false },
      { name: ROLE_NAMES.converter, word: `${up.device} toward SCADA · ${words.us} toward the device`, here: true },
      { name: DIAGRAM.device, word: words.device, here: false }]
    : [{ name: DIAGRAM.scada, word: words.us, here: false },
      { name: ROLE_NAMES.gateway, word: `${words.device} toward SCADA · IEC 61850 client toward the IEDs`, here: true },
      { name: DIAGRAM.ieds, word: 'IEC 61850 servers', here: false }];
  return <div className="side-diagram" aria-label={DIAGRAM.title}>
    <span className="small muted">{DIAGRAM.title}</span>
    {boxes.map((box, index) => <span key={box.name} className="row side-step">
      {index > 0 && <ArrowLeftRight size={16} aria-hidden />}
      <span className={`side-box${box.here ? ' here' : ''}`}>
        {box.here && <span className="side-here">{DIAGRAM.here}</span>}
        <strong>{box.name}</strong><span className="small">{box.word}</span>
      </span>
    </span>)}
  </div>;
}
