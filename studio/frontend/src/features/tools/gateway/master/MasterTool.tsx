import { useCallback, useEffect, useState } from 'react';
import { masterApi, type MasterConfig, type MasterProtocol, type MasterStatus } from '../../../../api/master';
import { MASTER_COPY, RANGES_COPY, TRAFFIC_WORDS } from '../../../../copy/master';
import { GATEWAY_COPY } from '../../../../copy/gateway';
import { TrafficLog } from '../TrafficLog';
import { MasterCommandLog } from './MasterCommandLog';
import { MasterConnect } from './MasterConnect';
import { MasterPointTable } from './MasterPointTable';
import { MasterStatusBar } from './MasterStatusBar';
import { ModbusRanges } from './ModbusRanges';
import '../../explorer/explorer.css';
import '../gateway.css';
import './master.css';

type Tab = 'points' | 'ranges' | 'traffic' | 'commands';

/** ScadaBench as SCADA master: connect to a device, discover every point, send confirmed commands. */
export function MasterTool() {
  const [config, setConfig] = useState<MasterConfig | null>(null);
  const [status, setStatus] = useState<MasterStatus | null>(null);
  const [tab, setTab] = useState<Tab>('points');
  const [busy, setBusy] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [error, setError] = useState('');
  const refresh = useCallback(() => masterApi.status().then(setStatus).catch(reason => setError(reason.message)), []);
  useEffect(() => {
    masterApi.config().then(setConfig).catch(reason => setError(reason.message));
    refresh();
    const timer = setInterval(refresh, 1500);
    return () => clearInterval(timer);
  }, [refresh]);
  const act = (task: () => Promise<unknown>) => {
    setBusy(true); setError('');
    task().catch(reason => setError(reason.message)).finally(() => { setBusy(false); refresh(); });
  };
  const save = (protocol: MasterProtocol, host: string, settings: Record<string, number | boolean>) =>
    act(() => masterApi.saveConfig(protocol, host, settings).then(setConfig));
  if (!config || !status) return <div className="card"><div className="guide-empty"><p>Loading the SCADA master…</p>
    {error && <p className="explorer-error">{error}</p>}</div></div>;
  const connected = status.state !== 'disconnected' && status.state !== 'failed';
  const disconnect = () => act(masterApi.disconnect);
  const modbus = config.protocol === 'modbus';
  const tabs: [Tab, string][] = [['points', GATEWAY_COPY.tabs.points], ...(modbus ? [['ranges', RANGES_COPY.tab] as [Tab, string]] : []),
    ['traffic', GATEWAY_COPY.tabs.traffic], ['commands', GATEWAY_COPY.tabs.commands]];
  return <div className={connected ? 'gateway-fill' : 'gateway-flow'}>
    {connected && !settingsOpen
      ? <MasterStatusBar status={status} busy={busy} disconnect={disconnect} showSettings={() => setSettingsOpen(true)}
        interrogate={() => act(masterApi.interrogate)} />
      : <MasterConnect config={config} status={status} busy={busy} save={save} disconnect={disconnect}
        connect={() => { setSettingsOpen(false); act(masterApi.connect); }}
        hide={connected ? () => setSettingsOpen(false) : undefined} />}
    {!status.available && <p className="explorer-error">The master program is not built in this copy of ScadaBench.</p>}
    {error && <p className="explorer-error" role="alert">{error}</p>}
    <div className={`card gateway-panel${connected ? ' running' : ''}`}>
      <div className="tabbar" role="tablist">{tabs.map(([key, label]) =>
        <button key={key} role="tab" aria-selected={tab === key} className={tab === key ? 'active' : ''}
          onClick={() => setTab(key)}>{label}</button>)}</div>
      {tab === 'points' && <MasterPointTable status={status} fail={setError} />}
      {tab === 'ranges' && modbus && <ModbusRanges status={status} fail={setError} />}
      {tab === 'traffic' && <TrafficLog running={connected} load={masterApi.traffic} words={TRAFFIC_WORDS}
        hint={MASTER_COPY.trafficHint} />}
      {tab === 'commands' && <MasterCommandLog />}
    </div>
  </div>;
}
