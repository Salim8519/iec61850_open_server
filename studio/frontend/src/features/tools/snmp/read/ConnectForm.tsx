import { Copy, PlugZap, RefreshCw, Unplug } from 'lucide-react';
import { useState } from 'react';
import { snmpApi, type SnmpConfig, type SnmpStatus, type SnmpTarget, type SnmpVersion } from '../../../../api/snmp';
import { READ_COPY } from '../../../../copy/snmpRead';
import { HelpTip } from '../../../../ui/HelpTip';
import { SegmentedControl } from '../../../../ui/SegmentedControl';
import { Spinner } from '../../../../ui/Spinner';
import { UserFields } from '../simulate/UsersEditor';

const EVERY: [number, string][] = [[1000, '1 second'], [2000, '2 seconds'], [5000, '5 seconds'], [10000, '10 seconds'],
  [30000, '30 seconds'], [60000, '1 minute']];

/** The device to read. While connected it shows the device, the re-read interval and the actions;
 *  otherwise the address, version and community or v3 user with Connect. */
export function ConnectForm({ status, config, refresh, fail }: {
  status: SnmpStatus; config: SnmpConfig; refresh: () => void; fail: (message: string) => void;
}) {
  const [target, setTarget] = useState<SnmpTarget>(config.reader.target);
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState('');
  const reader = status.reader;
  const open = reader.state !== 'disconnected' && reader.state !== 'failed';
  const set = (part: Partial<SnmpTarget>) => setTarget(current => ({ ...current, ...part }));
  const act = (task: () => Promise<unknown>) => {
    setBusy(true);
    fail('');
    setNote('');
    task().catch(reason => fail(reason.message)).finally(() => { setBusy(false); refresh(); });
  };
  const system = (oid: string) => reader.system.find(row => row.oid === oid)?.value ?? '';
  const sysName = system('1.3.6.1.2.1.1.5.0') || target.host;
  const twin = () => act(() => snmpApi.saveTwin(READ_COPY.twinName(sysName))
    .then(answer => setNote(READ_COPY.twinSaved(answer.device.name, answer.device.points.length))));
  const state = <p className={`small ${reader.state === 'connected' ? 'snmp-live' : reader.state === 'failed'
    ? 'explorer-error' : 'muted'}`} role="status">
    {(reader.state === 'connecting' || reader.state === 'walking') && <Spinner label="" inline />}
    {reader.state === 'walking' ? READ_COPY.walking(reader.walk_count) : READ_COPY.states[reader.state]}</p>;
  if (open) return <div className="side-section">
    <h3 className="side-title"><PlugZap size={18} aria-hidden />{sysName}</h3>
    {state}
    <p className="small muted snmp-target-text">{reader.target}</p>
    {system('1.3.6.1.2.1.1.1.0') && <p className="small snmp-sysdescr">{system('1.3.6.1.2.1.1.1.0')}</p>}
    {reader.problem && <p className="small gateway-warning">{reader.problem}</p>}
    {reader.message && <p className="small gateway-warning">{reader.message}</p>}
    {reader.state === 'connected' && <>
      <label><span className="label-row">{READ_COPY.every}<HelpTip text={READ_COPY.everyHint} /></span>
        <select aria-label={READ_COPY.every} value={reader.every_ms}
          onChange={event => act(() => snmpApi.setPoll(Number(event.target.value)))}>
          {EVERY.map(([ms, words]) => <option key={ms} value={ms}>{words}</option>)}</select></label>
      <span className="small muted">{READ_COPY.values(reader.values, reader.polled)}</span>
      <button type="button" disabled={busy} onClick={() => act(() => snmpApi.walk())}><RefreshCw size={16} />{READ_COPY.walkAgain}</button>
      <button type="button" disabled={busy} onClick={twin}><Copy size={16} />{READ_COPY.saveTwin}</button>
      {note && <p className="small snmp-live" role="status">✓ {note}</p>}
    </>}
    <button type="button" disabled={busy} onClick={() => act(snmpApi.disconnect)}><Unplug size={16} />{READ_COPY.disconnect}</button>
  </div>;
  return <form className="side-section" onSubmit={event => { event.preventDefault(); act(() => snmpApi.connect(target)); }}>
    <h3 className="side-title"><PlugZap size={18} aria-hidden />{READ_COPY.connectTitle}</h3>
    <label>{READ_COPY.address}<input value={target.host} spellCheck={false} placeholder="192.168.1.20"
      onChange={event => set({ host: event.target.value.trim() })} /></label>
    <label>{READ_COPY.port}<input inputMode="numeric" value={target.port}
      onChange={event => set({ port: Number(event.target.value) || 0 })} /></label>
    <label>{READ_COPY.version}<SegmentedControl label={READ_COPY.version} value={target.version} full
      onChange={(version: SnmpVersion) => set({ version })}
      options={(['v1', 'v2c', 'v3'] as SnmpVersion[]).map(value => ({ value, label: value }))} /></label>
    {target.version === 'v3'
      ? <fieldset className="snmp-fieldset"><legend>{READ_COPY.user}</legend>
        <UserFields user={target.user} change={user => set({ user })} /></fieldset>
      : <label><span className="label-row">{READ_COPY.community}<HelpTip text={READ_COPY.communityHint} /></span>
        <input aria-label={READ_COPY.community} value={target.community ?? ''}
          onChange={event => set({ community: event.target.value })} /></label>}
    <span className="row snmp-connect"><button className="primary" type="submit" disabled={busy || !target.host}>
      <PlugZap size={16} />{reader.state === 'failed' ? READ_COPY.reconnect : READ_COPY.connect}</button>
      <HelpTip text={READ_COPY.connectHint} /></span>
    {reader.state === 'failed' && state}
    {reader.message && <p className="small explorer-error">{reader.message}</p>}
  </form>;
}
