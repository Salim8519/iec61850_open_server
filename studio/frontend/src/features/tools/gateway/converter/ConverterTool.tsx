import { useCallback, useEffect, useState } from 'react';
import { createActionQueue } from '../actionQueue';
import { GATEWAY_CONFIG, GATEWAY_STATUS, gatewayApi, type GatewayConfig, type GatewayStatus, type Listen, type Protocol } from '../../../../api/gateway';
import { liveFeed } from '../../../../api/polls';
import { MASTER_CONFIG, MASTER_STATUS, masterApi, type MasterConfig, type MasterProtocol, type MasterStatus } from '../../../../api/master';
import { CONVERTER_COPY } from '../../../../copy/converter';
import { MASTER_COPY, TRAFFIC_WORDS } from '../../../../copy/master';
import { TabBar } from '../../../../ui/TabBar';
import { FULL_PLACE } from '../../../../copy/fullView';
import { FullView, FullViewButton } from '../../../../ui/FullView';
import { useFullView } from '../../../../state/useFullView';
import { CommandLog } from '../CommandLog';
import { GatewaySettings } from '../GatewaySettings';
import { MasterConnect } from '../master/MasterConnect';
import { MasterPointTable } from '../master/MasterPointTable';
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

/** Protocol converter: ScadaBench reads the device as master and serves the same points upstream.
 *  The column holds "1 · Read the device" above "2 · Serve to SCADA", each with its own status and button. */
export function ConverterTool() {
  const [tab, setTab] = useState<Tab>('device');
  const [full, setFull] = useFullView('converter');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<{ side: 'device' | 'scada'; text: string } | null>(null);
  // Both sides' settings and status on the shared live feed, every 1.5 s as before (speed lane F).
  const [parts, setParts] = useState<{ mc?: MasterConfig; ms?: MasterStatus; gc?: GatewayConfig; gs?: GatewayStatus }>({});
  useEffect(() => {
    const fail = (reason: Error) => setError({ side: 'device', text: reason.message });
    const stops = [
      liveFeed.subscribe<MasterConfig>(MASTER_CONFIG, 1500, mc => setParts(current => ({ ...current, mc })), fail),
      liveFeed.subscribe<MasterStatus>(MASTER_STATUS, 1500, ms => setParts(current => ({ ...current, ms })), fail),
      liveFeed.subscribe<GatewayConfig>(GATEWAY_CONFIG, 1500, gc => setParts(current => ({ ...current, gc })), fail),
      liveFeed.subscribe<GatewayStatus>(GATEWAY_STATUS, 1500, gs => setParts(current => ({ ...current, gs })), fail),
    ];
    return () => stops.forEach(stop => stop());
  }, []);
  const refresh = useCallback(() => [MASTER_CONFIG, MASTER_STATUS, GATEWAY_CONFIG, GATEWAY_STATUS].forEach(liveFeed.refresh), []);
  const master = parts.mc && parts.ms ? { config: parts.mc, status: parts.ms } : null;
  const gateway = parts.gc && parts.gs ? { config: parts.gc, status: parts.gs } : null;
  const [queue] = useState(createActionQueue);
  // Saves run quietly so Connect or Start pressed right after typing a setting is not lost (one queue).
  const act = (side: 'device' | 'scada', task: () => Promise<unknown>, quiet = false) => {
    if (!quiet) setBusy(true);
    setError(null);
    queue(task).catch(reason => setError({ side, text: reason.message })).finally(() => { if (!quiet) setBusy(false); refresh(); });
  };
  const fail = (text: string) => setError({ side: 'device', text });
  if (!master || !gateway) return <div className="card fill-body"><div className="guide-empty"><p>Loading the protocol converter…</p>
    {error && <p className="explorer-error">{error.text}</p>}</div></div>;
  const connected = master.status.state === 'connected' || master.status.state === 'lost';
  const serving = gateway.status.state === 'running';
  return <div className="fill-body fill-split scada-split">
    <aside className="card side-pane" aria-label={CONVERTER_COPY.columnTitle}>
      <MasterConnect config={master.config} status={master.status} busy={busy} title={CONVERTER_COPY.deviceStep}
        help={`${CONVERTER_COPY.intro} ${MASTER_COPY.intro(master.config.protocol)}`}
        save={(protocol: MasterProtocol, host, settings) => act('device', () => masterApi.saveConfig(protocol, host, settings), true)}
        connect={() => act('device', masterApi.connect)} disconnect={() => act('device', masterApi.disconnect)}
        interrogate={() => act('device', masterApi.interrogate)} error={error?.side === 'device' ? error.text : ''} />
      <GatewaySettings config={gateway.config} status={gateway.status} busy={busy} words={WORDS} title={CONVERTER_COPY.scadaStep}
        save={(protocol: Protocol, listen: Listen, settings, real) => act('scada', () => gatewayApi.saveConfig(protocol, listen, settings, real), true)}
        start={() => act('scada', gatewayApi.start)} stop={() => act('scada', gatewayApi.stop)}
        error={error?.side === 'scada' ? error.text : ''} />
    </aside>
    <FullView full={full} change={setFull} place={FULL_PLACE.converter}>
    <div className={`card fill-panel gateway-panel converter-panel${serving ? ' running' : ''}`}>
      <TabBar label="Converter views" value={tab} change={setTab} end={<FullViewButton full={full} change={setFull} />}
        tabs={(Object.keys(CONVERTER_COPY.tabs) as Tab[]).map(key => [key, CONVERTER_COPY.tabs[key]])} />
      <div className="panel-body">
        {tab === 'device' && <MasterPointTable status={master.status} fail={fail} />}
        {tab === 'served' && <ServedPoints protocol={gateway.config.protocol} fail={fail} />}
        {tab === 'routes' && <CommandRoutes locked={serving} fail={fail} />}
        {tab === 'scada' && <TrafficLog running={serving} />}
        {tab === 'deviceTraffic' && <TrafficLog running={connected} load={masterApi.traffic} feedKey="master-traffic" words={TRAFFIC_WORDS}
          hint={MASTER_COPY.trafficHint} />}
        {tab === 'commands' && <CommandLog protocol={gateway.config.protocol} />}
      </div>
    </div>
    </FullView>
  </div>;
}
