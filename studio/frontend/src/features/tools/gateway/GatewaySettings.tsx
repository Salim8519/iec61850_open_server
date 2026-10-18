import { Play, Square } from 'lucide-react';
import type { ReactNode } from 'react';
import type { GatewayConfig, GatewayStatus, Listen, Protocol } from '../../../api/gateway';
import { ADVANCED_SETTINGS, GATEWAY_COPY, MAIN_SETTINGS, PROTOCOL_HINTS, PROTOCOL_NAMES, SETTING_LABELS } from '../../../copy/gateway';
import { Disclosure } from '../../../ui/Disclosure';
import { HelpTip } from '../../../ui/HelpTip';
import { StatusPill } from '../../../ui/StatusPill';

/** The settings column of the gateway (ADR-0064): protocol, who may connect, the few settings SCADA
 *  must match, then status, errors and Start/Stop at the bottom. Long explanations sit behind "?". */
export function GatewaySettings({ config, status, busy, save, start, stop, words, title, error, children }: {
  config: GatewayConfig; status: GatewayStatus; busy: boolean;
  save: (protocol: Protocol, listen: Listen, settings: Record<string, number | boolean>, realCommands: boolean) => void;
  start: () => void; stop: () => void; words?: Partial<typeof GATEWAY_COPY>; title?: string; error?: string;
  children?: ReactNode;
}) {
  const copy = { ...GATEWAY_COPY, ...words };
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
    : { label: 'Stopped', tone: 'off' as const, hint: copy.startHint };
  return <section className="side-section" aria-label={title ?? copy.settingsTitle}>
    <h3 className="side-title">{title ?? copy.settingsTitle}<HelpTip text={copy.intro} label="What does this do?" /></h3>
    {children}
    <label><span className="label-row">Protocol<HelpTip text={PROTOCOL_HINTS[protocol]} label="About this protocol" /></span>
      <select value={protocol} disabled={running || busy}
        onChange={event => { const next = event.target.value as Protocol; save(next, config.listen, config.settings[next], real); }}>
        {(Object.keys(PROTOCOL_NAMES) as Protocol[]).map(value => <option key={value} value={value}>{PROTOCOL_NAMES[value]}</option>)}
      </select></label>
    <label><span className="label-row">{copy.listen}<HelpTip text={copy.listenHint} label="Who can connect?" /></span>
      <select value={config.listen} disabled={running || busy} onChange={event => save(protocol, event.target.value as Listen, values, real)}>
        <option value="this_pc">{copy.thisPc}</option><option value="network">{copy.network}</option></select></label>
    {MAIN_SETTINGS[protocol].map(field)}
    <span className="label-row"><label className="row gateway-check"><input type="checkbox" checked={real} disabled={running || busy}
      onChange={event => save(protocol, config.listen, values, event.target.checked)} />{copy.realCommands}</label>
      <HelpTip text={real ? copy.realCommandsOn : copy.realCommandsOff} label="What does this allow?" /></span>
    {ADVANCED_SETTINGS[protocol].length > 0 && <Disclosure>{ADVANCED_SETTINGS[protocol].map(field)}</Disclosure>}
    <div className="side-foot">
      <StatusPill status={pill} />
      {running && <p className="side-line" aria-live="polite">{status.message} · {status.clients.length
        ? `${copy.connected(status.clients.length)}: ${status.clients.join(', ')}` : copy.none}</p>}
      {!running && status.state === 'failed' && status.message && <p className="side-error" role="alert">{status.message}</p>}
      {config.auto_start && !running && <p className="side-line">{copy.autoStart}</p>}
      {error && <p className="side-error" role="alert">{error}</p>}
      {running
        ? <button onClick={stop} disabled={busy}><Square size={16} />{copy.stop}</button>
        : <button className="primary" onClick={start} disabled={busy || !status.available}><Play size={16} />{copy.start}</button>}
    </div>
  </section>;
}
