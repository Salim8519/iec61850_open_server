import { ChevronUp, PlugZap, Unplug } from 'lucide-react';
import type { MasterConfig, MasterProtocol, MasterStatus } from '../../../../api/master';
import { PROTOCOL_NAMES } from '../../../../copy/gateway';
import { MASTER_ADVANCED, MASTER_COPY, MASTER_MAIN, MASTER_SETTING_LABELS } from '../../../../copy/master';
import { Disclosure } from '../../../../ui/Disclosure';
import { SegmentedControl } from '../../../../ui/SegmentedControl';
import { StatusPill } from '../../../../ui/StatusPill';

const AVAILABLE: MasterProtocol[] = ['iec104', 'dnp3', 'modbus'];
const FLAGS = new Set(['time_tagged_commands', 'unsolicited']);

/** Device address, protocol and the few settings that must match the device; Connect/Disconnect. */
export function MasterConnect({ config, status, busy, save, connect, disconnect, hide }: {
  config: MasterConfig; status: MasterStatus; busy: boolean;
  save: (protocol: MasterProtocol, host: string, settings: Record<string, number | boolean>) => void;
  connect: () => void; disconnect: () => void; hide?: () => void;
}) {
  const connected = status.state === 'connected' || status.state === 'lost';
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
  return <div className="card">
    <div className="guide-section">
      <p className="muted" style={{ margin: 0 }}>{MASTER_COPY.intro(protocol)}</p>
      <div className="row gateway-top">
        <SegmentedControl label="Protocol" value={protocol} disabled={connected || busy}
          options={AVAILABLE.map(value => ({ value, label: PROTOCOL_NAMES[value] }))}
          onChange={next => save(next, config.host, config.settings[next])} />
        <span className="gateway-spacer" />
        <StatusPill status={{ label: MASTER_COPY.states[status.state], tone, hint: status.message }} />
        {hide && <button className="ghost" onClick={hide}><ChevronUp size={16} />Hide settings</button>}
        {connected
          ? <button onClick={disconnect} disabled={busy}><Unplug size={16} />{MASTER_COPY.disconnect}</button>
          : <button className="primary" onClick={connect} disabled={busy || !status.available || !config.host}>
            <PlugZap size={16} />{MASTER_COPY.connect}</button>}
      </div>
      <div className="form-grid gateway-fields">
        <label>{MASTER_COPY.host}<input defaultValue={config.host} key={`host-${config.host}`} disabled={connected}
          placeholder="192.168.1.20" aria-describedby="master-host-hint"
          onBlur={event => { const next = event.target.value.trim(); if (next !== config.host) save(protocol, next, values); }} /></label>
        {MASTER_MAIN[protocol].map(field)}
      </div>
      <p className="small muted" id="master-host-hint" style={{ margin: 0 }}>{MASTER_COPY.hostHint} {MASTER_COPY.intervalHint[protocol]} {MASTER_COPY.allowList}</p>
      {status.state !== 'disconnected' && status.message && <p className="small gateway-warning" role="status" style={{ margin: 0 }}>
        {status.message}</p>}
      {config.auto_connect && !connected && <p className="small muted" style={{ margin: 0 }}>{MASTER_COPY.autoConnect}</p>}
      {MASTER_ADVANCED[protocol].length > 0 && <Disclosure><div className="form-grid gateway-fields">
        {MASTER_ADVANCED[protocol].map(field)}</div></Disclosure>}
    </div>
  </div>;
}
