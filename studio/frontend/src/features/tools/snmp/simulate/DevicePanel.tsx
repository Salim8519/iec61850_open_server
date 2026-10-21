import { Trash2 } from 'lucide-react';
import { useState } from 'react';
import { snmpApi, type DeviceStatus, type SnmpDevice } from '../../../../api/snmp';
import { SETTINGS_COPY, SNMP_COPY } from '../../../../copy/snmp';
import { TabBar } from '../../../../ui/TabBar';
import { DeviceSettings } from './DeviceSettings';
import { PointsTable } from './PointsTable';
import { RequestLog } from './RequestLog';
import { TrapPanel } from './TrapPanel';

type Tab = 'values' | 'settings' | 'traps' | 'log';

/** One simulated device: its values, settings, traps and the requests it answered. */
export function DevicePanel({ device, state, refresh, fail }: {
  device: SnmpDevice; state?: DeviceStatus; refresh: () => void; fail: (message: string) => void;
}) {
  const [tab, setTab] = useState<Tab>('values');
  const running = state?.state === 'running';
  const save = (next: SnmpDevice) => snmpApi.update(next).then(refresh);
  const remove = () => {
    if (!window.confirm(SNMP_COPY.removeConfirm(device.name))) return;
    snmpApi.remove(device.id).then(refresh).catch(reason => fail(reason.message));
  };
  return <div className={`card fill-panel gateway-panel${running ? ' running' : ''}`}>
    <div className="panel-head">
      <h3>{device.name}</h3>
      <span className="small muted">{state?.kind} · {state?.address ?? `${device.ip}:${device.port}`}
        {state?.network && ` · ${SETTINGS_COPY.onNetwork}`}</span>
      <span className="gateway-spacer" />
      <button className="ghost" onClick={remove}><Trash2 size={16} />{SNMP_COPY.remove}</button>
    </div>
    <TabBar label="Device views" value={tab} change={setTab}
      tabs={(Object.keys(SNMP_COPY.tabs) as Tab[]).map(key => [key, SNMP_COPY.tabs[key]])} />
    <div className="panel-body">
      {tab === 'values' && <PointsTable device={device} running={running} refresh={refresh} fail={fail} />}
      {tab === 'settings' && <DeviceSettings device={device} state={state} running={running} save={save} fail={fail} />}
      {tab === 'traps' && <TrapPanel device={device} running={running} save={save} fail={fail} />}
      {tab === 'log' && <RequestLog device={device.id} />}
    </div>
  </div>;
}
