import { useEffect, useState } from 'react';
import type { Activity } from '../../../api/activity';
import { masterApi, type MasterProtocol, type Role } from '../../../api/master';
import { ConverterTool } from './converter/ConverterTool';
import { GatewayTool } from './GatewayTool';
import { MasterTool } from './master/MasterTool';
import { RoleSwitch } from './RoleSwitch';

/** "SCADA gateway & master": one tool, a role switch, and the view of the chosen role (ADR-0061). */
export function ScadaTool({ runState, runIeds, activity }: {
  runState: string; runIeds: () => void; activity: Activity | null;
}) {
  const [role, setRole] = useState<Role | null>(null);
  const [error, setError] = useState('');
  useEffect(() => { masterApi.config().then(config => setRole(config.role)).catch(reason => setError(reason.message)); }, []);
  const busy = activity?.gateway.state === 'running' || (activity?.master && activity.master.state !== 'disconnected');
  const change = (next: Role) => {
    setError('');
    masterApi.setRole(next).then(answer => setRole(answer.role)).catch(reason => setError(reason.message));
  };
  if (!role) return error ? <p className="explorer-error" role="alert">{error}</p> : null;
  const protocol = (activity?.master.protocol || 'iec104') as MasterProtocol;
  const upstream = (activity?.gateway.protocol || 'iec104') as MasterProtocol;
  return <div className="scada-tool">
    <RoleSwitch role={role} locked={Boolean(busy)} change={change} protocol={protocol} upstream={upstream} />
    {error && <p className="explorer-error" role="alert">{error}</p>}
    {role === 'master' ? <MasterTool /> : role === 'converter' ? <ConverterTool />
      : <GatewayTool runState={runState} runIeds={runIeds} />}
  </div>;
}
