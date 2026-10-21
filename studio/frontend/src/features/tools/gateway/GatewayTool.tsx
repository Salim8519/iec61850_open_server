import { useCallback, useEffect, useState } from 'react';
import { createActionQueue } from './actionQueue';
import { GATEWAY_STATUS, gatewayApi, type GatewayConfig, type GatewayStatus, type Listen, type Protocol } from '../../../api/gateway';
import { liveFeed } from '../../../api/polls';
import type { Ied } from '../../../api/types';
import { GATEWAY_COPY } from '../../../copy/gateway';
import { TabBar } from '../../../ui/TabBar';
import { CommandLog } from './CommandLog';
import { GatewayFirewall } from './GatewayFirewall';
import { GatewayNotices } from './GatewayNotices';
import { GatewaySettings } from './GatewaySettings';
import { PointTable } from './PointTable';
import { TrafficLog } from './TrafficLog';
import '../explorer/explorer.css';
import './gateway.css';

type Tab = 'points' | 'traffic' | 'commands';

/** Gateway simulator: ScadaBench as the station gateway towards a SCADA master. Run stays separate.
 *  Settings column on the left, the tabs fill the rest of the window (ADR-0064). */
export function GatewayTool({ ieds, runState, runIeds }: { ieds: Ied[]; runState: string; runIeds: () => void }) {
  const [config, setConfig] = useState<GatewayConfig | null>(null);
  const [status, setStatus] = useState<GatewayStatus | null>(null);
  const [tab, setTab] = useState<Tab>('points');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const refresh = useCallback(() => liveFeed.refresh(GATEWAY_STATUS), []);
  useEffect(() => {
    gatewayApi.config().then(setConfig).catch(reason => setError(reason.message));
    return liveFeed.subscribe<GatewayStatus>(GATEWAY_STATUS, 1500, setStatus, reason => setError(reason.message));
  }, []);
  const [queue] = useState(createActionQueue);
  // Saves run quietly so a Start pressed right after typing a setting is not lost (they share the queue).
  const act = (task: () => Promise<unknown>, quiet = false) => {
    if (!quiet) setBusy(true);
    setError('');
    queue(task).catch(reason => setError(reason.message)).finally(() => { if (!quiet) setBusy(false); refresh(); });
  };
  const save = (protocol: Protocol, listen: Listen, settings: Record<string, number | boolean>, realCommands: boolean) =>
    act(() => gatewayApi.saveConfig(protocol, listen, settings, realCommands).then(setConfig), true);
  if (!config || !status) return <div className="card fill-body"><div className="guide-empty"><p>Loading the gateway…</p>
    {error && <p className="explorer-error">{error}</p>}</div></div>;
  const running = status.state === 'running';
  return <div className="fill-body fill-split scada-split">
    <aside className="card side-pane" aria-label={GATEWAY_COPY.settingsTitle}>
      <GatewaySettings config={config} status={status} busy={busy} save={save} stop={() => act(gatewayApi.stop)}
        start={() => act(gatewayApi.start)} error={error || (status.available ? '' : GATEWAY_COPY.notBuilt)}>
        <GatewayNotices status={status} runState={runState} runIeds={runIeds} />
        {config.listen === 'network' && <GatewayFirewall port={Number(config.settings[config.protocol].port)} />}
      </GatewaySettings>
    </aside>
    <div className={`card fill-panel gateway-panel${running ? ' running' : ''}`}>
      <TabBar label="Gateway views" value={tab} change={setTab}
        tabs={(Object.keys(GATEWAY_COPY.tabs) as Tab[]).map(key => [key, GATEWAY_COPY.tabs[key]])} />
      <div className="panel-body">
        {tab === 'points' && <PointTable protocol={config.protocol} ieds={ieds} locked={running} fail={setError} />}
        {tab === 'traffic' && <TrafficLog running={running} />}
        {tab === 'commands' && <CommandLog />}
      </div>
    </div>
  </div>;
}
