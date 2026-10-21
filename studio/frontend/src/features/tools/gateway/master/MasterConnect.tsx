import { PlugZap, RefreshCw, Unplug } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { MasterConfig, MasterProtocol, MasterStatus } from '../../../../api/master';
import { PROTOCOL_NAMES } from '../../../../copy/gateway';
import { ASK_AGAIN, MASTER_ADVANCED, MASTER_COPY, MASTER_MAIN, MASTER_SETTING_LABELS, RANGES_COPY } from '../../../../copy/master';
import { Disclosure } from '../../../../ui/Disclosure';
import { HelpTip } from '../../../../ui/HelpTip';
import { Spinner } from '../../../../ui/Spinner';
import { StatusPill } from '../../../../ui/StatusPill';

const AVAILABLE: MasterProtocol[] = ['iec104', 'dnp3', 'modbus'];
const FLAGS = new Set(['time_tagged_commands', 'unsolicited']);

/** The device side in the settings column (ADR-0064): protocol, address and the few settings that
 *  must match the device; status, the device's answer and Connect/Disconnect at the bottom. */
export function MasterConnect({ config, status, busy, save, connect, disconnect, interrogate, title, help, error }: {
  config: MasterConfig; status: MasterStatus; busy: boolean;
  save: (protocol: MasterProtocol, host: string, settings: Record<string, number | boolean>) => void;
  connect: () => void; disconnect: () => void; interrogate: () => void; title?: string; help?: string; error?: string;
}) {
  const connected = status.state === 'connected' || status.state === 'lost';
  // A connect can wait up to t0 (10 s) for the device: say so instead of showing a silent "Not connected".
  const [connecting, setConnecting] = useState(false);
  useEffect(() => { if (!busy) setConnecting(false); }, [busy]);
  const protocol = config.protocol;
  const values = config.settings[protocol];
  const change = (key: string, value: number | boolean) => save(protocol, config.host, { ...values, [key]: value });
  const field = (key: string) => FLAGS.has(key)
    ? <label key={key} className="row gateway-check"><input type="checkbox" checked={Boolean(values[key])} disabled={connected}
        onChange={event => change(key, event.target.checked)} />{MASTER_SETTING_LABELS[key]}</label>
    : <label key={key}>{MASTER_SETTING_LABELS[key]}
        <input inputMode="numeric" defaultValue={String(values[key])} key={`${protocol}-${key}-${values[key]}`} disabled={connected}
          onBlur={event => { const next = Number(event.target.value); if (next !== values[key]) change(key, next); }} /></label>;
  const tone = status.state === 'connected' ? 'good' as const : status.state === 'disconnected' ? 'off' as const : 'problem' as const;
  const name = title ?? MASTER_COPY.settingsTitle;
  return <section className="side-section" aria-label={name}>
    <h3 className="side-title">{name}<HelpTip text={help ?? MASTER_COPY.intro(protocol)} label="What does this do?" /></h3>
    <label>Protocol<select value={protocol} disabled={connected || busy}
      onChange={event => { const next = event.target.value as MasterProtocol; save(next, config.host, config.settings[next]); }}>
      {AVAILABLE.map(value => <option key={value} value={value}>{PROTOCOL_NAMES[value]}</option>)}</select></label>
    <label><span className="label-row">{MASTER_COPY.host}
      <HelpTip text={`${MASTER_COPY.hostHint} ${MASTER_COPY.allowList}`} label="Which address?" /></span>
      <input defaultValue={config.host} key={`host-${config.host}`} disabled={connected} placeholder="192.168.1.20"
        onBlur={event => { const next = event.target.value.trim(); if (next !== config.host) save(protocol, next, values); }} /></label>
    {MASTER_MAIN[protocol].map(field)}
    <p className="side-line">{MASTER_COPY.intervalHint[protocol]}</p>
    {MASTER_ADVANCED[protocol].length > 0 && <Disclosure>{MASTER_ADVANCED[protocol].map(field)}</Disclosure>}
    <div className="side-foot">
      {connecting && !connected ? <Spinner label={MASTER_COPY.connecting(config.host, values.port)} slowHint={MASTER_COPY.connectingSlow} />
        : <StatusPill status={{ label: MASTER_COPY.states[status.state], tone, hint: status.message }} />}
      {connected && <p className="side-line">{status.host}:{status.port} · {MASTER_COPY.points(status.points)}</p>}
      {status.state !== 'disconnected' && status.state !== 'connected' && status.message &&
        <p className="side-error" role="status">{status.message}</p>}
      {status.poll_failures.length > 0 && <p className="side-error">{RANGES_COPY.refused(status.poll_failures.length)}</p>}
      {config.auto_connect && !connected && <p className="side-line">{MASTER_COPY.autoConnect}</p>}
      {error && <p className="side-error" role="alert">{error}</p>}
      {connected && <button onClick={interrogate} disabled={busy || status.state !== 'connected'} title={ASK_AGAIN[protocol].hint}>
        <RefreshCw size={16} />{ASK_AGAIN[protocol].label}</button>}
      {connected
        ? <button onClick={disconnect} disabled={busy}><Unplug size={16} />{MASTER_COPY.disconnect}</button>
        : <button className="primary" onClick={() => { setConnecting(true); connect(); }} disabled={busy || !status.available || !config.host}>
          <PlugZap size={16} />{MASTER_COPY.connect}</button>}
    </div>
  </section>;
}
