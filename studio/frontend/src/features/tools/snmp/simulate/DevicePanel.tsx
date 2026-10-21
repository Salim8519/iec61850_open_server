import { MoreHorizontal, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { snmpApi, type DeviceStatus, type SnmpDevice } from '../../../../api/snmp';
import { SETTINGS_COPY, SNMP_COPY } from '../../../../copy/snmp';
import { Menu } from '../../../../ui/Menu';
import { TabBar } from '../../../../ui/TabBar';
import { SNMP_FULL_PLACE } from '../../../../copy/fullView';
import { FullView, FullViewButton } from '../../../../ui/FullView';
import { useFullView } from '../../../../state/useFullView';
import { DeviceSettings } from './DeviceSettings';
import { PointsTable } from './PointsTable';
import { RequestLog } from './RequestLog';
import { TrapPanel } from './TrapPanel';

type Tab = 'values' | 'settings' | 'traps' | 'log';

/** One simulated device: its values, settings, traps and the requests it answered. The device's name and
 *  address, the tabs and the "⋯" menu (Delete device, asked again before it happens) share one row. */
export function DevicePanel({ device, state, refresh, fail }: {
  device: SnmpDevice; state?: DeviceStatus; refresh: () => void; fail: (message: string) => void;
}) {
  const [tab, setTab] = useState<Tab>('values');
  const [full, setFull] = useFullView('snmpSimulate');
  const running = state?.state === 'running';
  const save = (next: SnmpDevice) => snmpApi.update(next).then(refresh);
  const remove = () => {
    if (!window.confirm(SNMP_COPY.removeConfirm(device.name))) return;
    snmpApi.remove(device.id).then(refresh).catch(reason => fail(reason.message));
  };
  return <FullView full={full} change={setFull} place={`${SNMP_FULL_PLACE.simulate} › ${device.name}`}
    detail={state?.address ?? `${device.ip}:${device.port}`}>
  <div className={`card fill-panel gateway-panel${running ? ' running' : ''}`}>
    <div className="panel-head snmp-device-head">
      <h3>{device.name}</h3>
      <span className="small muted snmp-device-address">{state?.kind} · {state?.address ?? `${device.ip}:${device.port}`}
        {state?.network && ` · ${SETTINGS_COPY.onNetwork}`}</span>
      <TabBar label="Device views" value={tab} change={setTab}
        tabs={(Object.keys(SNMP_COPY.tabs) as Tab[]).map(key => [key, SNMP_COPY.tabLabel(key, SNMP_COPY.tabs[key], device.points.length)])} />
      <span className="gateway-spacer" />
      <FullViewButton full={full} change={setFull} />
      <Menu label={SNMP_COPY.deviceMenu} trigger={<MoreHorizontal size={18} aria-hidden />} triggerClass="icon"
        items={[{ label: SNMP_COPY.remove, icon: <Trash2 size={16} />, onSelect: remove }]} />
    </div>
    <div className="panel-body">
      {tab === 'values' && <PointsTable device={device} running={running} refresh={refresh} fail={fail} />}
      {tab === 'settings' && <DeviceSettings device={device} state={state} running={running} save={save} fail={fail} />}
      {tab === 'traps' && <TrapPanel device={device} running={running} save={save} fail={fail} />}
      {tab === 'log' && <RequestLog device={device.id} />}
    </div>
  </div>
  </FullView>;
}
