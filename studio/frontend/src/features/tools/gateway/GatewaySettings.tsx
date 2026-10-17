import { Play, Square } from 'lucide-react';
import type { GatewayConfig, GatewayStatus, Listen, Protocol } from '../../../api/gateway';
import { ADVANCED_SETTINGS, GATEWAY_COPY, MAIN_SETTINGS, PROTOCOL_HINTS, PROTOCOL_NAMES, SETTING_LABELS } from '../../../copy/gateway';
import { Disclosure } from '../../../ui/Disclosure';
import { SegmentedControl } from '../../../ui/SegmentedControl';
import { StatusPill } from '../../../ui/StatusPill';

/** Protocol, who may connect, the few settings SCADA must match, and Start/Stop. */
export function GatewaySettings({ config, status, busy, save, start, stop }: {
  config: GatewayConfig; status: GatewayStatus; busy: boolean;
  save: (protocol: Protocol, listen: Listen, settings: Record<string, number | boolean>, realCommands: boolean) => void;
  start: () => void; stop: () => void;
}) {
  const running = status.state === 'running';
  const protocol = config.protocol;
  const values = config.settings[protocol];
  const real = config.real_commands;
  const change = (key: string, value: number | boolean) => save(protocol, config.listen, { ...values, [key]: value }, real);
  const field = (key: string) => key === 'unsolicited'
    ? <label key={key} className="row gateway-check"><input type="checkbox" checked={Boolean(values[key])} disabled={running}
        onChange={event => change(key, event.target.checked)} />{SETTING_LABELS[key]}</label>
    : <label key={key}>{SETTING_LABELS[key]}
        <input inputMode="numeric" defaultValue={String(values[key])} key={`${protocol}-${key}-${values[key]}`} disabled={running}
          onBlur={event => { const next = Number(event.target.value); if (next !== values[key]) change(key, next); }} /></label>;
  const pill = running
    ? { label: 'Running', tone: 'good' as const, hint: status.message }
    : status.state === 'failed' ? { label: 'Stopped with a problem', tone: 'problem' as const, hint: status.message }
    : { label: 'Stopped', tone: 'off' as const, hint: GATEWAY_COPY.startHint };
  return <div className="card">
    <div className="guide-section">
      <p className="muted" style={{ margin: 0 }}>{GATEWAY_COPY.intro}</p>
      <div className="row gateway-top">
        <SegmentedControl label="Protocol" value={protocol} disabled={running || busy}
          options={(Object.keys(PROTOCOL_NAMES) as Protocol[]).map(value => ({ value, label: PROTOCOL_NAMES[value] }))}
          onChange={next => save(next, config.listen, config.settings[next], real)} />
        <SegmentedControl label={GATEWAY_COPY.listen} value={config.listen} disabled={running || busy}
          options={[{ value: 'this_pc', label: GATEWAY_COPY.thisPc }, { value: 'network', label: GATEWAY_COPY.network }]}
          onChange={next => save(protocol, next as Listen, values, real)} />
        <span className="gateway-spacer" />
        <StatusPill status={pill} />
        {running
          ? <button onClick={stop} disabled={busy}><Square size={16} />{GATEWAY_COPY.stop}</button>
          : <button className="primary" onClick={start} disabled={busy || !status.available}><Play size={16} />{GATEWAY_COPY.start}</button>}
      </div>
      <p className="small muted" style={{ margin: 0 }}>{PROTOCOL_HINTS[protocol]} {config.listen === 'network' ? GATEWAY_COPY.listenHint : ''}</p>
      <div className="form-grid gateway-fields">{MAIN_SETTINGS[protocol].map(field)}</div>
      <label className="row gateway-check"><input type="checkbox" checked={real} disabled={running || busy}
        onChange={event => save(protocol, config.listen, values, event.target.checked)} />{GATEWAY_COPY.realCommands}</label>
      <p className="small muted" style={{ margin: 0 }}>{real ? GATEWAY_COPY.realCommandsOn : GATEWAY_COPY.realCommandsOff}</p>
      {config.auto_start && !running && <p className="small muted" style={{ margin: 0 }}>{GATEWAY_COPY.autoStart}</p>}
      {ADVANCED_SETTINGS[protocol].length > 0 && <Disclosure><div className="form-grid gateway-fields">
        {ADVANCED_SETTINGS[protocol].map(field)}</div></Disclosure>}
      {running && <p className="gateway-live" aria-live="polite"><strong>{status.message}</strong> ·{' '}
        {status.clients.length ? `${GATEWAY_COPY.connected(status.clients.length)}: ${status.clients.join(', ')}` : GATEWAY_COPY.none}</p>}
      {status.run_state === 'stopped' && <p className="small gateway-warning">{GATEWAY_COPY.notRunning}</p>}
      {status.problems.length > 0 && <ul className="small gateway-warning">{status.problems.slice(0, 5).map(problem =>
        <li key={problem}>Skipped: {problem}</li>)}</ul>}
    </div>
  </div>;
}
