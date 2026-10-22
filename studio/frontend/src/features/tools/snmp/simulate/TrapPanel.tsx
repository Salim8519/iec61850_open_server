import { Plus, Send, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { snmpApi, type Action, type SnmpDevice, type SnmpVersion, type TrapResult, type TrapTarget } from '../../../../api/snmp';
import { SETTINGS_COPY, SNMP_COPY, TRAPS_COPY } from '../../../../copy/snmp';
import { HelpTip } from '../../../../ui/HelpTip';
import { SaveProblem, SaveStatus } from '../../../../ui/SaveStatus';
import { Spinner } from '../../../../ui/Spinner';
import type { DeviceDraft } from './useDeviceDraft';

const NEW_TARGET: TrapTarget = { host: '127.0.0.1', port: 162, version: 'v2c', community: 'public' };

/** Where the device sends traps, and the events that send one (link down, GPS lost, restart…). The
 *  receivers are saved as they change, together with the settings (one draft, see useDeviceDraft). */
export function TrapPanel({ device, running, settings, fail }: {
  device: SnmpDevice; running: boolean; settings: DeviceDraft; fail: (message: string) => void;
}) {
  const { draft, set, problem, refused, state } = settings;
  const targets = draft.traps;
  const [actions, setActions] = useState<Action[]>([]);
  const [action, setAction] = useState('');
  const [sending, setSending] = useState(false);
  const [results, setResults] = useState<TrapResult[]>([]);
  useEffect(() => {
    snmpApi.actions(device.id).then(answer => { setActions(answer.actions); setAction(answer.actions[0]?.id ?? ''); })
      .catch(reason => fail(reason.message));
  }, [device.id, fail]);
  const change = (index: number, part: Partial<TrapTarget>, typing = false) =>
    set({ traps: targets.map((target, at) => at === index ? { ...target, ...part } : target) }, typing);
  const unsaved = problem !== null || state.phase === 'saving' || state.phase === 'failed';
  const send = () => {
    setSending(true);
    setResults([]);
    fail('');
    snmpApi.trap(device.id, action).then(answer => setResults(answer.results)).catch(reason => fail(reason.message))
      .finally(() => setSending(false));
  };
  const users = draft.users.map(user => user.name);
  return <div className="fill-scroll snmp-settings" {...settings.edge}>
    <fieldset className="snmp-fieldset"><legend><span className="label-row">{TRAPS_COPY.targets}
      <HelpTip text={TRAPS_COPY.targetsHint} /></span></legend>
      {targets.map((target, index) => <div key={index} className="row snmp-target">
        <label>{TRAPS_COPY.host}<input value={target.host} onChange={event => change(index, { host: event.target.value.trim() }, true)} /></label>
        <label>{TRAPS_COPY.port}<input className="gateway-number" inputMode="numeric" value={target.port || ''}
          onChange={event => change(index, { port: Number(event.target.value.replace(/\D/g, '')) || 0 }, true)} /></label>
        <label>Version<select value={target.version} onChange={event => change(index, { version: event.target.value as SnmpVersion,
          inform: event.target.value === 'v1' ? false : target.inform, user: target.user || users[0] })}>
          <option value="v1">v1</option><option value="v2c">v2c</option><option value="v3">v3</option></select></label>
        {target.version === 'v3'
          ? <label>{TRAPS_COPY.user}<select value={target.user} onChange={event => change(index, { user: event.target.value })}>
            {users.map(name => <option key={name} value={name}>{name}</option>)}</select></label>
          : <label>{TRAPS_COPY.community}<input value={target.community ?? ''}
            onChange={event => change(index, { community: event.target.value }, true)} /></label>}
        {target.version !== 'v1' && <label className="row gateway-check" title={TRAPS_COPY.informHint}>
          <input type="checkbox" checked={Boolean(target.inform)} onChange={event => change(index, { inform: event.target.checked })} />
          {TRAPS_COPY.inform}</label>}
        <button type="button" className="ghost icon" aria-label="Remove receiver"
          onClick={() => set({ traps: targets.filter((_, at) => at !== index) })}><Trash2 size={14} /></button>
      </div>)}
      <div className="row">
        <button type="button" onClick={() => set({ traps: [...targets, NEW_TARGET] })}>
          <Plus size={16} />{TRAPS_COPY.addTarget}</button>
        <SaveStatus state={state} />
        {running && <span className="small muted">{SETTINGS_COPY.autoRunning}</span>}
      </div>
      <SaveProblem text={problem?.field === 'traps' ? problem.text : problem ? SETTINGS_COPY.settingsProblem(problem.text) : refused} />
    </fieldset>
    <fieldset className="snmp-fieldset"><legend><span className="label-row">{TRAPS_COPY.event}
      <HelpTip text={TRAPS_COPY.sendHint} /></span></legend>
      <div className="row">
        <select aria-label={TRAPS_COPY.event} value={action} onChange={event => setAction(event.target.value)}>
          {actions.map(item => <option key={item.id} value={item.id}>{item.label}</option>)}</select>
        <button className="primary" disabled={!running || sending || unsaved || targets.length === 0 || !action} onClick={send}>
          <Send size={16} />{TRAPS_COPY.send}</button>
        {sending && <Spinner label="Sending…" inline />}
      </div>
      {!running && <p className="small muted">{SNMP_COPY.startHint}</p>}
      {targets.length === 0 && <p className="small muted">{TRAPS_COPY.noTargets}</p>}
      {results.length > 0 && <ul className="snmp-results" aria-label={TRAPS_COPY.results}>
        {results.map(result => <li key={result.target} className={result.ok ? 'quality-good' : 'quality-bad'}>
          {result.ok ? '✓ ' : '⚠ '}{result.target}: {result.text}</li>)}</ul>}
    </fieldset>
  </div>;
}
