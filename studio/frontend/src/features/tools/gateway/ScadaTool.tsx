import { useEffect, useState, type ReactNode } from 'react';
import type { Activity } from '../../../api/activity';
import { masterApi, type MasterProtocol, type Role } from '../../../api/master';
import type { Ied } from '../../../api/types';
import { ConverterTool } from './converter/ConverterTool';
import { GatewayTool } from './GatewayTool';
import { MasterTool } from './master/MasterTool';
import { RoleSwitch } from './RoleSwitch';

/** "SCADA gateway & master": one tool, a role switch in the header row, and the view of the chosen
 *  role filling the window (ADR-0061, ADR-0064). */
export function ScadaTool({ header, ieds, runState, runIeds, activity }: {
  header: (extra?: ReactNode) => ReactNode; ieds: Ied[]; runState: string; runIeds: () => void; activity: Activity | null;
}) {
  const [role, setRole] = useState<Role | null>(null);
  const [error, setError] = useState('');
  useEffect(() => { masterApi.config().then(config => setRole(config.role)).catch(reason => setError(reason.message)); }, []);
  const busy = activity?.gateway.state === 'running' || (activity?.master && activity.master.state !== 'disconnected');
  const change = (next: Role) => {
    setError('');
    masterApi.setRole(next).then(answer => setRole(answer.role)).catch(reason => setError(reason.message));
  };
  const protocol = (activity?.master.protocol || 'iec104') as MasterProtocol;
  const upstream = (activity?.gateway.protocol || 'iec104') as MasterProtocol;
  return <>
    {header(role && <RoleSwitch role={role} locked={Boolean(busy)} change={change} protocol={protocol} upstream={upstream} />)}
    {error && <p className="explorer-error" role="alert">{error}</p>}
    {role === 'master' ? <MasterTool /> : role === 'converter' ? <ConverterTool />
      : role === 'gateway' ? <GatewayTool ieds={ieds} runState={runState} runIeds={runIeds} /> : null}
  </>;
}
