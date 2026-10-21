import { useCallback, useEffect, useState } from 'react';
import { createActionQueue } from '../actionQueue';
import { MASTER_STATUS, masterApi, type MasterConfig, type MasterProtocol, type MasterStatus } from '../../../../api/master';
import { liveFeed } from '../../../../api/polls';
import { MASTER_COPY, RANGES_COPY, TRAFFIC_WORDS } from '../../../../copy/master';
import { GATEWAY_COPY } from '../../../../copy/gateway';
import { TabBar } from '../../../../ui/TabBar';
import { FULL_PLACE } from '../../../../copy/fullView';
import { FullView, FullViewButton } from '../../../../ui/FullView';
import { useFullView } from '../../../../state/useFullView';
import { TrafficLog } from '../TrafficLog';
import { MasterCommandLog } from './MasterCommandLog';
import { MasterConnect } from './MasterConnect';
import { MasterPointTable } from './MasterPointTable';
import { ModbusRanges } from './ModbusRanges';
import '../../explorer/explorer.css';
import '../gateway.css';
import './master.css';

type Tab = 'points' | 'ranges' | 'traffic' | 'commands';

/** ScadaBench as SCADA master: connect to a device, discover every point, send confirmed commands.
 *  Connection column on the left, the tabs fill the rest of the window (ADR-0064). */
export function MasterTool() {
  const [config, setConfig] = useState<MasterConfig | null>(null);
  const [status, setStatus] = useState<MasterStatus | null>(null);
  const [tab, setTab] = useState<Tab>('points');
  const [full, setFull] = useFullView('master');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const refresh = useCallback(() => liveFeed.refresh(MASTER_STATUS), []);
  useEffect(() => {
    masterApi.config().then(setConfig).catch(reason => setError(reason.message));
    return liveFeed.subscribe<MasterStatus>(MASTER_STATUS, 1500, setStatus, reason => setError(reason.message));
  }, []);
  const [queue] = useState(createActionQueue);
  // Saves run quietly so a Connect pressed right after typing the address is not lost (they share the queue).
  const act = (task: () => Promise<unknown>, quiet = false) => {
    if (!quiet) setBusy(true);
    setError('');
    queue(task).catch(reason => setError(reason.message)).finally(() => { if (!quiet) setBusy(false); refresh(); });
  };
  const save = (protocol: MasterProtocol, host: string, settings: Record<string, number | boolean>) =>
    act(() => masterApi.saveConfig(protocol, host, settings).then(setConfig), true);
  if (!config || !status) return <div className="card fill-body"><div className="guide-empty"><p>Loading the SCADA master…</p>
    {error && <p className="explorer-error">{error}</p>}</div></div>;
  const connected = status.state !== 'disconnected' && status.state !== 'failed';
  const modbus = config.protocol === 'modbus';
  const tabs: [Tab, string][] = [['points', GATEWAY_COPY.tabs.points], ...(modbus ? [['ranges', RANGES_COPY.tab] as [Tab, string]] : []),
    ['traffic', GATEWAY_COPY.tabs.traffic], ['commands', GATEWAY_COPY.tabs.commands]];
  return <div className="fill-body fill-split scada-split">
    <aside className="card side-pane" aria-label={MASTER_COPY.settingsTitle}>
      <MasterConnect config={config} status={status} busy={busy} save={save} disconnect={() => act(masterApi.disconnect)}
        connect={() => act(masterApi.connect)} interrogate={() => act(masterApi.interrogate)}
        error={error || (status.available ? '' : MASTER_COPY.notBuilt)} />
    </aside>
    <FullView full={full} change={setFull} place={FULL_PLACE.master} detail={`${config.protocol.toUpperCase()} · ${status.state}`}>
    <div className={`card fill-panel gateway-panel${connected ? ' running' : ''}`}>
      <TabBar label="Master views" value={tab} change={setTab} tabs={tabs} end={<FullViewButton full={full} change={setFull} />} />
      <div className="panel-body">
        {tab === 'points' && <MasterPointTable status={status} fail={setError} />}
        {tab === 'ranges' && modbus && <ModbusRanges status={status} fail={setError} />}
        {tab === 'traffic' && <TrafficLog running={connected} load={masterApi.traffic} feedKey="master-traffic" words={TRAFFIC_WORDS}
          hint={MASTER_COPY.trafficHint} />}
        {tab === 'commands' && <MasterCommandLog />}
      </div>
    </div>
    </FullView>
  </div>;
}
