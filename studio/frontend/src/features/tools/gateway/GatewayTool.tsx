import { useCallback, useEffect, useState } from 'react';
import { gatewayApi, type GatewayConfig, type GatewayStatus, type Listen, type Protocol } from '../../../api/gateway';
import { GATEWAY_COPY } from '../../../copy/gateway';
import { CommandLog } from './CommandLog';
import { GatewayFirewall } from './GatewayFirewall';
import { GatewayNotices } from './GatewayNotices';
import { GatewaySettings } from './GatewaySettings';
import { GatewayStatusBar } from './GatewayStatusBar';
import { PointTable } from './PointTable';
import { TrafficLog } from './TrafficLog';
import '../explorer/explorer.css';
import './gateway.css';

type Tab = 'points' | 'traffic' | 'commands';

/** Gateway simulator: ScadaBench as the station gateway towards a SCADA master. Run stays separate. */
export function GatewayTool({ runState, runIeds }: { runState: string; runIeds: () => void }) {
  const [config, setConfig] = useState<GatewayConfig | null>(null);
  const [status, setStatus] = useState<GatewayStatus | null>(null);
  const [tab, setTab] = useState<Tab>('points');
  const [busy, setBusy] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [error, setError] = useState('');
  const refresh = useCallback(() => gatewayApi.status().then(setStatus).catch(reason => setError(reason.message)), []);
  useEffect(() => {
    gatewayApi.config().then(setConfig).catch(reason => setError(reason.message));
    refresh();
    const timer = setInterval(refresh, 1500);
    return () => clearInterval(timer);
  }, [refresh]);
  const act = (task: () => Promise<unknown>) => {
    setBusy(true); setError('');
    task().catch(reason => setError(reason.message)).finally(() => { setBusy(false); refresh(); });
  };
  const save = (protocol: Protocol, listen: Listen, settings: Record<string, number | boolean>, realCommands: boolean) =>
    act(() => gatewayApi.saveConfig(protocol, listen, settings, realCommands).then(setConfig));
  if (!config || !status) return <div className="card"><div className="guide-empty"><p>Loading the gateway…</p>
    {error && <p className="explorer-error">{error}</p>}</div></div>;
  const running = status.state === 'running';
  const stop = () => act(gatewayApi.stop);
  return <div className={running ? 'gateway-fill' : 'gateway-flow'}>
    {running && !settingsOpen
      ? <GatewayStatusBar status={status} busy={busy} stop={stop} showSettings={() => setSettingsOpen(true)} />
      : <GatewaySettings config={config} status={status} busy={busy} save={save} stop={stop}
        start={() => { setSettingsOpen(false); act(gatewayApi.start); }}
        hide={running ? () => setSettingsOpen(false) : undefined} />}
    <GatewayNotices status={status} runState={runState} runIeds={runIeds} />
    {config.listen === 'network' && <div className="card"><div className="guide-section">
      <GatewayFirewall port={Number(config.settings[config.protocol].port)} /></div></div>}
    {!status.available && <p className="explorer-error">The gateway program is not built in this copy of ScadaBench.</p>}
    {error && <p className="explorer-error" role="alert">{error}</p>}
    <div className={`card gateway-panel${running ? ' running' : ''}`}>
      <div className="tabbar" role="tablist">{(Object.keys(GATEWAY_COPY.tabs) as Tab[]).map(key =>
        <button key={key} role="tab" aria-selected={tab === key} className={tab === key ? 'active' : ''}
          onClick={() => setTab(key)}>{GATEWAY_COPY.tabs[key]}</button>)}</div>
      {tab === 'points' && <PointTable protocol={config.protocol} locked={running} fail={setError} />}
      {tab === 'traffic' && <TrafficLog running={running} />}
      {tab === 'commands' && <CommandLog />}
    </div>
  </div>;
}
