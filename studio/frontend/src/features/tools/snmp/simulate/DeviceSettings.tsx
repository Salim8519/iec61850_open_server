import { useState } from 'react';
import type { DeviceStatus } from '../../../../api/snmp';
import { SETTINGS_COPY } from '../../../../copy/snmp';
import { Disclosure } from '../../../../ui/Disclosure';
import { HelpTip } from '../../../../ui/HelpTip';
import { SaveProblem, SaveStatus } from '../../../../ui/SaveStatus';
import type { DeviceProblem } from './deviceRules';
import type { DeviceDraft } from './useDeviceDraft';
import { UsersEditor } from './UsersEditor';

const HERE: DeviceProblem['field'][] = ['name', 'ip', 'port', 'versions', 'community', 'users', 'engine'];

/** A device's address, versions, communities, v3 users and engine ID. Every valid change is saved at once
 *  (the status says Saving… then Saved); a wrong value keeps its text and shows one red line under it. */
export function DeviceSettings({ state, running, settings }: { state?: DeviceStatus; running: boolean; settings: DeviceDraft }) {
  const { draft, set, problem, refused } = settings;
  const address = state?.network ? SETTINGS_COPY.localAddress : SETTINGS_COPY.address;
  const [portText, setPortText] = useState(String(draft.port));
  const at = (field: DeviceProblem['field']) => problem?.field === field ? problem.text : '';
  const elsewhere = problem && !HERE.includes(problem.field) ? SETTINGS_COPY.trapProblem(problem.text) : '';
  return <form className="fill-scroll snmp-settings" onSubmit={event => event.preventDefault()} {...settings.edge}>
    <p className="row small muted snmp-autosave"><SaveStatus state={settings.state} />
      {running ? SETTINGS_COPY.autoRunning : SETTINGS_COPY.autoNote}</p>
    <div className="form-grid">
      <div><label>{SETTINGS_COPY.name}<input value={draft.name} onChange={event => set({ name: event.target.value }, true)} /></label>
        <SaveProblem text={at('name')} /></div>
      <div><label><span className="label-row">{address}<HelpTip text={SETTINGS_COPY.addressHint} /></span>
        <input aria-label={address} value={draft.ip} spellCheck={false}
          onChange={event => set({ ip: event.target.value.trim() }, true)} /></label><SaveProblem text={at('ip')} /></div>
      <div><label><span className="label-row">{SETTINGS_COPY.port}<HelpTip text={SETTINGS_COPY.portHint} /></span>
        <input aria-label={SETTINGS_COPY.port} inputMode="numeric" value={portText} onChange={event => {
          setPortText(event.target.value);
          set({ port: /^\d{1,5}$/.test(event.target.value.trim()) ? Number(event.target.value) : 0 }, true);
        }} /></label><SaveProblem text={at('port')} /></div>
    </div>
    {state?.network && <p className="small snmp-network-note">{SETTINGS_COPY.networkNote(state.address)}</p>}
    <fieldset className="snmp-fieldset"><legend><span className="label-row">{SETTINGS_COPY.versions}
      <HelpTip text={SETTINGS_COPY.versionsHint} /></span></legend>
      <div className="row">{(['v1', 'v2c', 'v3'] as const).map(version => <label key={version} className="row gateway-check">
        <input type="checkbox" checked={draft[version]} onChange={event => set({ [version]: event.target.checked })} />
        {version}</label>)}</div>
      <SaveProblem text={at('versions')} />
    </fieldset>
    {(draft.v1 || draft.v2c) && <div className="form-grid">
      <div><label><span className="label-row">{SETTINGS_COPY.readCommunity}<HelpTip text={SETTINGS_COPY.communityHint} /></span>
        <input aria-label={SETTINGS_COPY.readCommunity} value={draft.read_community}
          onChange={event => set({ read_community: event.target.value }, true)} /></label><SaveProblem text={at('community')} /></div>
      <div><label>{SETTINGS_COPY.writeCommunity}
        <input value={draft.write_community} onChange={event => set({ write_community: event.target.value }, true)} /></label></div>
    </div>}
    {draft.v3 && <UsersEditor users={draft.users} change={(users, typing) => set({ users }, typing)} problem={at('users')} />}
    <Disclosure>
      <label><span className="label-row">{SETTINGS_COPY.engineId}<HelpTip text={SETTINGS_COPY.engineHint} /></span>
        <input aria-label={SETTINGS_COPY.engineId} className="mono" value={draft.engine_id} spellCheck={false}
          onChange={event => set({ engine_id: event.target.value.trim() }, true)} /></label>
    </Disclosure>
    <SaveProblem text={at('engine') || elsewhere || refused} />
  </form>;
}
