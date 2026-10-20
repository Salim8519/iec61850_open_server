import { CircleStop, Network, Play, Plus } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { snmpApi, type SnmpConfig, type SnmpStatus } from '../../../../api/snmp';
import { SETTINGS_COPY, SNMP_COPY } from '../../../../copy/snmp';
import { HelpTip } from '../../../../ui/HelpTip';
import { Menu } from '../../../../ui/Menu';
import { DevicePanel } from './DevicePanel';
import { ImportDevice } from './ImportDevice';

type Importing = 'walk' | 'csv' | null;

/** Simulate devices: the device list with Start/Stop on the left, the chosen device on the right. */
export function SimulateView({ status, config, refresh, fail }: {
  status: SnmpStatus; config: SnmpConfig; refresh: () => void; fail: (message: string) => void;
}) {
  const [selected, setSelected] = useState<string>(() => config.devices[0]?.id ?? '');
  const [busy, setBusy] = useState(false);
  const [importing, setImporting] = useState<Importing>(null);
  const known = useRef(config.devices.map(device => device.id));
  useEffect(() => {
    const fresh = config.devices.find(device => !known.current.includes(device.id));
    known.current = config.devices.map(device => device.id);
    if (fresh) setSelected(fresh.id); // a device just added or imported opens at once
    else if (!config.devices.some(device => device.id === selected)) setSelected(config.devices[0]?.id ?? '');
  }, [config.devices, selected]);
  const act = (task: () => Promise<unknown>) => {
    setBusy(true);
    fail('');
    task().catch(reason => fail(reason.message)).finally(() => { setBusy(false); refresh(); });
  };
  const add = (profile: string) => act(() => snmpApi.add(profile));
  const running = status.running > 0;
  const device = config.devices.find(item => item.id === selected);
  return <div className="fill-body fill-split scada-split">
    <aside className="card side-pane snmp-side" aria-label={SNMP_COPY.devices}>
      <h3 className="side-title"><Network size={18} aria-hidden />{SNMP_COPY.devices}</h3>
      <div className="row snmp-run">
        {running
          ? <button className="danger" disabled={busy} onClick={() => act(snmpApi.stop)}><CircleStop size={16} />{SNMP_COPY.stop}</button>
          : <button className="primary" disabled={busy || config.devices.length === 0} onClick={() => act(snmpApi.start)}>
            <Play size={16} />{SNMP_COPY.start}</button>}
        <HelpTip text={SNMP_COPY.startHint} label="What does Start do?" />
      </div>
      <p className={`small ${running ? 'snmp-live' : 'muted'}`} role="status">
        {running && <span className="live-dot" aria-hidden />}{running ? SNMP_COPY.running(status.running) : SNMP_COPY.stopped}</p>
      <Menu label={SNMP_COPY.addDevice} align="left" triggerClass="snmp-add"
        trigger={<><Plus size={16} />{SNMP_COPY.addDevice}</>} items={[
          { label: SNMP_COPY.addMenu.switch, onSelect: () => add('switch') },
          { label: SNMP_COPY.addMenu.gps, onSelect: () => add('gps') },
          { label: SNMP_COPY.addMenu.empty, onSelect: () => add('empty') },
          { label: SNMP_COPY.addMenu.walk, onSelect: () => setImporting('walk') },
          { label: SNMP_COPY.addMenu.csv, onSelect: () => setImporting('csv') },
        ]} />
      {importing && <ImportDevice kind={importing} close={() => setImporting(null)} done={refresh} fail={fail} />}
      <ul className="snmp-device-list" aria-label={SNMP_COPY.devices}>
        {status.devices.map(item => <li key={item.id}>
          <button type="button" className={`snmp-device${item.id === selected ? ' active' : ''}`} aria-pressed={item.id === selected}
            onClick={() => setSelected(item.id)}>
            <strong>{item.name}</strong>
            <span className="small muted">{item.kind || item.profile} · {item.address}</span>
            {item.network && <span className="small muted">{SETTINGS_COPY.onNetworkLine}</span>}
            <span className={`small snmp-state state-${item.state}`}>
              {item.state === 'running' ? '● ' : item.state === 'failed' ? '⚠ ' : '○ '}{SNMP_COPY.states[item.state]}</span>
            {item.error && <span className="small explorer-error">{item.error}</span>}
          </button></li>)}
      </ul>
    </aside>
    {device ? <DevicePanel key={device.id} device={device} state={status.devices.find(item => item.id === device.id)}
      refresh={refresh} fail={fail} />
      : <div className="card fill-panel"><div className="panel-empty"><div className="guide-empty">
        <span className="guide-empty-icon"><Network size={24} /></span>
        <h3>{config.devices.length ? SNMP_COPY.pickDevice : SNMP_COPY.noDevices}</h3>
        {!config.devices.length && <p>{SNMP_COPY.noDevicesHint}</p>}</div></div></div>}
  </div>;
}
