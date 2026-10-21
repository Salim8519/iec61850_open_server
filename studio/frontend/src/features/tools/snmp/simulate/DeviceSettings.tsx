import { Save } from 'lucide-react';
import { useState } from 'react';
import type { DeviceStatus, SnmpDevice } from '../../../../api/snmp';
import { SETTINGS_COPY, SNMP_COPY } from '../../../../copy/snmp';
import { Disclosure } from '../../../../ui/Disclosure';
import { HelpTip } from '../../../../ui/HelpTip';
import { UsersEditor } from './UsersEditor';

/** A device's address, versions, communities, v3 users and engine ID; saved together with Save settings. */
export function DeviceSettings({ device, state, running, save, fail }: {
  device: SnmpDevice; state?: DeviceStatus; running: boolean; save: (device: SnmpDevice) => Promise<unknown>;
  fail: (message: string) => void;
}) {
  const address = state?.network ? SETTINGS_COPY.localAddress : SETTINGS_COPY.address;
  const [draft, setDraft] = useState(device);
  const [saved, setSaved] = useState(false);
  const set = (change: Partial<SnmpDevice>) => { setDraft(current => ({ ...current, ...change })); setSaved(false); };
  const submit = () => {
    fail('');
    save(draft).then(() => setSaved(true)).catch(reason => fail(reason.message));
  };
  return <form className="fill-scroll snmp-settings" onSubmit={event => { event.preventDefault(); submit(); }}>
    <div className="form-grid">
      <label>{SETTINGS_COPY.name}<input value={draft.name} onChange={event => set({ name: event.target.value })} /></label>
      <label><span className="label-row">{address}<HelpTip text={SETTINGS_COPY.addressHint} /></span>
        <input aria-label={address} value={draft.ip} onChange={event => set({ ip: event.target.value.trim() })} spellCheck={false} /></label>
      <label><span className="label-row">{SETTINGS_COPY.port}<HelpTip text={SETTINGS_COPY.portHint} /></span>
        <input aria-label={SETTINGS_COPY.port} inputMode="numeric" value={draft.port} onChange={event => set({ port: Number(event.target.value) || 0 })} /></label>
    </div>
    {state?.network && <p className="small snmp-network-note">{SETTINGS_COPY.networkNote(state.address)}</p>}
    <fieldset className="snmp-fieldset"><legend><span className="label-row">{SETTINGS_COPY.versions}
      <HelpTip text={SETTINGS_COPY.versionsHint} /></span></legend>
      <div className="row">{(['v1', 'v2c', 'v3'] as const).map(version => <label key={version} className="row gateway-check">
        <input type="checkbox" checked={draft[version]} onChange={event => set({ [version]: event.target.checked })} />
        {version}</label>)}</div>
    </fieldset>
    {(draft.v1 || draft.v2c) && <div className="form-grid">
      <label><span className="label-row">{SETTINGS_COPY.readCommunity}<HelpTip text={SETTINGS_COPY.communityHint} /></span>
        <input aria-label={SETTINGS_COPY.readCommunity} value={draft.read_community} onChange={event => set({ read_community: event.target.value })} /></label>
      <label>{SETTINGS_COPY.writeCommunity}
        <input value={draft.write_community} onChange={event => set({ write_community: event.target.value })} /></label>
    </div>}
    {draft.v3 && <UsersEditor users={draft.users} change={users => set({ users })} />}
    <Disclosure>
      <label><span className="label-row">{SETTINGS_COPY.engineId}<HelpTip text={SETTINGS_COPY.engineHint} /></span>
        <input aria-label={SETTINGS_COPY.engineId} className="mono" value={draft.engine_id} onChange={event => set({ engine_id: event.target.value.trim() })}
          spellCheck={false} /></label>
    </Disclosure>
    <div className="row">
      <button className="primary" type="submit"><Save size={16} />{SETTINGS_COPY.save}</button>
      {saved && <span className="small snmp-live">✓ {SETTINGS_COPY.saved}</span>}
      {running && <span className="small muted">{SNMP_COPY.restartNote}</span>}
    </div>
  </form>;
}
