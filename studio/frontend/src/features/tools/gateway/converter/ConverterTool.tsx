import { useCallback, useEffect, useState } from 'react';
import { gatewayApi, type GatewayConfig, type GatewayStatus, type Listen, type Protocol } from '../../../../api/gateway';
import { masterApi, type MasterConfig, type MasterProtocol, type MasterStatus } from '../../../../api/master';
import { CONVERTER_COPY } from '../../../../copy/converter';
import { MASTER_COPY, TRAFFIC_WORDS } from '../../../../copy/master';
import { CommandLog } from '../CommandLog';
import { GatewaySettings } from '../GatewaySettings';
import { GatewayStatusBar } from '../GatewayStatusBar';
import { MasterConnect } from '../master/MasterConnect';
import { MasterPointTable } from '../master/MasterPointTable';
import { MasterStatusBar } from '../master/MasterStatusBar';
import { TrafficLog } from '../TrafficLog';
import { CommandRoutes } from './CommandRoutes';
import { ServedPoints } from './ServedPoints';
import '../../explorer/explorer.css';
import '../gateway.css';
import '../master/master.css';

type Tab = keyof typeof CONVERTER_COPY.tabs;
const WORDS = { intro: CONVERTER_COPY.intro, realCommands: CONVERTER_COPY.realCommands, start: CONVERTER_COPY.start,
  stop: CONVERTER_COPY.stop, realCommandsOn: CONVERTER_COPY.realCommandsOn, realCommandsOff: CONVERTER_COPY.realCommandsOff,
  startHint: CONVERTER_COPY.startHint };

/** Protocol converter: ScadaBench reads the device as master and serves the same points upstream. */
export function ConverterTool() {
  const [master, setMaster] = useState<{ config: MasterConfig; status: MasterStatus } | null>(null);
  const [gateway, setGateway] = useState<{ config: GatewayConfig; status: GatewayStatus } | null>(null);
  const [tab, setTab] = useState<Tab>('device');
  const [open, setOpen] = useState({ device: false, scada: false });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const refresh = useCallback(() => Promise.all([masterApi.config(), masterApi.status(), gatewayApi.config(), gatewayApi.status()])
    .then(([mc, ms, gc, gs]) => { setMaster({ config: mc, status: ms }); setGateway({ config: gc, status: gs }); })
    .catch(reason => setError(reason.message)), []);
  useEffect(() => { refresh(); const timer = setInterval(refresh, 1500); return () => clearInterval(timer); }, [refresh]);
  const act = (task: () => Promise<unknown>) => {
    setBusy(true); setError('');
    task().catch(reason => setError(reason.message)).finally(() => { setBusy(false); refresh(); });
  };
  if (!master || !gateway) return <div className="card"><div className="guide-empty"><p>Loading the protocol converter…</p>
    {error && <p className="explorer-error">{error}</p>}</div></div>;
  const connected = master.status.state === 'connected' || master.status.state === 'lost';
  const serving = gateway.status.state === 'running';
  return <div className={connected || serving ? 'gateway-fill' : 'gateway-flow'}>
    {connected && !open.device
      ? <MasterStatusBar status={master.status} busy={busy} disconnect={() => act(masterApi.disconnect)}
        showSettings={() => setOpen({ ...open, device: true })} interrogate={() => act(masterApi.interrogate)} />
      : <MasterConnect config={master.config} status={master.status} busy={busy} disconnect={() => act(masterApi.disconnect)}
        save={(protocol: MasterProtocol, host, settings) => act(() => masterApi.saveConfig(protocol, host, settings))}
        connect={() => act(masterApi.connect)} hide={connected ? () => setOpen({ ...open, device: false }) : undefined} />}
    {serving && !open.scada
      ? <GatewayStatusBar status={gateway.status} busy={busy} stop={() => act(gatewayApi.stop)} stopLabel={CONVERTER_COPY.stop}
        showSettings={() => setOpen({ ...open, scada: true })} />
      : <GatewaySettings config={gateway.config} status={gateway.status} busy={busy} words={WORDS}
        save={(protocol: Protocol, listen: Listen, settings, real) => act(() => gatewayApi.saveConfig(protocol, listen, settings, real))}
        start={() => act(gatewayApi.start)} stop={() => act(gatewayApi.stop)}
        hide={serving ? () => setOpen({ ...open, scada: false }) : undefined} />}
    {error && <p className="explorer-error" role="alert">{error}</p>}
    <div className={`card gateway-panel${serving ? ' running' : ''}`}>
      <div className="tabbar" role="tablist">{(Object.keys(CONVERTER_COPY.tabs) as Tab[]).map(key =>
        <button key={key} role="tab" aria-selected={tab === key} className={tab === key ? 'active' : ''}
          onClick={() => setTab(key)}>{CONVERTER_COPY.tabs[key]}</button>)}</div>
      {tab === 'device' && <MasterPointTable status={master.status} fail={setError} />}
      {tab === 'served' && <ServedPoints protocol={gateway.config.protocol} fail={setError} />}
      {tab === 'routes' && <CommandRoutes locked={serving} fail={setError} />}
      {tab === 'scada' && <TrafficLog running={serving} />}
      {tab === 'deviceTraffic' && <TrafficLog running={connected} load={masterApi.traffic} words={TRAFFIC_WORDS}
        hint={MASTER_COPY.trafficHint} />}
      {tab === 'commands' && <CommandLog />}
    </div>
  </div>;
}
