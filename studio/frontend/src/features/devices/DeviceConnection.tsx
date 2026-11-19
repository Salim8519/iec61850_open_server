import { Save, ShieldCheck } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { LiveSession } from '../../api/network';
import type { Ied, LiveSettings } from '../../api/types';
import { hint } from '../../copy/hints';
import { liveStatus } from '../../copy/status';
import { HelpTip } from '../../ui/HelpTip';
import { SegmentedControl } from '../../ui/SegmentedControl';
import { StatusPill } from '../../ui/StatusPill';
import { withCommands } from './allowCommands';
import { DeviceCommandCheck } from './DeviceCommandCheck';
import { draftKey } from './liveDraft';

/** How ScadaBench reaches a real IED: it reads values here; gateway commands need Allow commands (ADR-0138). */
export function DeviceConnection({ ied, session, locked, perform }: {
  ied: Ied; session?: LiveSession; locked: boolean; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const [draft, setDraft] = useState<LiveSettings>(ied.live);
  const saved = draftKey(ied.id, ied.live);
  // Reset only for another IED or new saved values, not for every project read (fix plan A17).
  useEffect(() => setDraft(ied.live), [saved]);
  const scd = ied.scl_address?.ip || '';
  const changed = JSON.stringify(draft) !== JSON.stringify(ied.live);
  return <div className="stack">
    {session ? <div className="mode-banner"><div className="stack" style={{ gap: 6 }}>
      <StatusPill status={liveStatus(session.state)} showHint />
      <span className="small">{session.message}</span>
      <span className="small muted">Device address <code>{session.host}:{session.port}</code></span>
      {session.access?.notes.map(note => <span key={note} className="small row" style={{ alignItems: 'flex-start' }}>
        <ShieldCheck size={15} style={{ flexShrink: 0, color: 'var(--success)' }} />{note}</span>)}
    </div></div> : <p className="muted" style={{ margin: 0 }}>{hint('live.not_running')} {hint('live.commands')}</p>}
    {session && <DeviceCommandCheck iedId={ied.id} live={session.state === 'live'} />}
    <div className="form-grid">
      <label>Device address<input value={draft.ip} disabled={locked} placeholder={scd || 'e.g. 192.168.1.20'}
        onChange={event => setDraft({ ...draft, ip: event.target.value.trim() })} />
        <span className="small muted" style={{ fontWeight: 400 }}>{draft.ip ? 'Typed by you' : scd ? `From the SCD file: ${scd}` : 'Not in the SCD file'}</span></label>
      <label>Port<input type="number" min={1} max={65535} value={draft.port}
        disabled={locked} onChange={event => setDraft({ ...draft, port: Number(event.target.value) })} /></label>
    </div>
    <div className="stack" style={{ gap: 8 }}>
      <span className="section-label">How ScadaBench reads values</span>
      <SegmentedControl label="How ScadaBench reads values" value={draft.access} disabled={locked}
        onChange={access => setDraft({ ...draft, access })} options={[
          { value: 'auto_reports', label: 'Reports (recommended)' }, { value: 'polling', label: 'Ask every second' }]} />
      <span className="small muted">{hint(`live.access.${draft.access}`)}</span>
    </div>
    <div className="stack" style={{ gap: 8 }}>
      <span className="label-row"><label className="row" style={{ gap: 8, fontWeight: 600 }}>
        <input type="checkbox" checked={!!draft.allow_commands} disabled={locked}
          onChange={event => setDraft(withCommands(draft, event.target.checked))} />Allow commands</label>
        <HelpTip text={hint('live.commands')} label="Where do commands come from?" /></span>
      <span className="small muted">{hint(draft.allow_commands ? 'live.allow_commands.on' : 'live.allow_commands.off')}</span>
    </div>
    <div className="row"><button className="primary" disabled={locked || !changed}
      onClick={() => perform(() => api(`/ieds/${ied.id}/settings`, 'PUT', { live: draft }))}><Save size={16} />Save connection</button>
      {locked && <span className="small muted">{hint('live.locked')}</span>}</div>
  </div>;
}
