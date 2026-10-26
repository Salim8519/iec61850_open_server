import { CircleAlert, ExternalLink, Network, Radio, ShieldCheck, TriangleAlert } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import { liveFeed } from '../../api/polls';
import type { GooseShare } from '../../api/network';
import { REACH_COPY, REACH_STATES, reachStatus } from '../../copy/gooseReach';
import { Disclosure } from '../../ui/Disclosure';
import { Hint } from '../../ui/Hint';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import { GooseShareWarning } from './GooseShareWarning';
import './goose.css';

const SHARE = '/goose/share';
const NPCAP_URL = 'https://npcap.com/#download';
// States that carry their own reason from the Go engine; shown under the choice.
const WITH_REASON = new Set(['confirm', 'share_held', 'checking', 'listening']);

/** "Where GOOSE goes" (ADR-0119): with the IEDs' network by default, or only inside ScadaBench.
 * Every decision comes from the Go engine; this card only shows it and sends the user's choice. */
export function GooseReach({ revision, perform }: { revision: number; perform: (task: () => Promise<unknown>) => Promise<void> }) {
  const [share, setShare] = useState<GooseShare | null>(null);
  const [asking, setAsking] = useState(false);
  const [busy, setBusy] = useState(false);
  const live = !!share?.running;
  // Shared live feed: on open and each revision, then every 2 s while GOOSE runs.
  useEffect(() => liveFeed.subscribe<GooseShare>(SHARE, live ? 2000 : null, setShare), [revision, live]);
  if (!share) return null;
  const send = (body: Record<string, unknown>) => {
    setBusy(true);
    return perform(() => api<GooseShare>(SHARE, 'PUT', body).then(setShare))
      .finally(() => { setBusy(false); setAsking(false); liveFeed.refresh(SHARE); });
  };
  const local = share.reach === 'local';
  const adapter = share.uplink.guid ? share.uplink.label : '';
  const waiting = share.state === 'confirm' || share.state === 'share_held';
  // The red Npcap line carries the Go engine's own words (ADR-0119).
  const npcapMissing = share.goes_out && !share.npcap && share.state === 'share_held';
  const status = !share.running && share.goes_out && !waiting ? REACH_STATES.goes_at_run : reachStatus(share.state);
  const choice = (value: 'auto' | 'local', title: string, line: string) =>
    <label className={`goose-reach-option${share.reach === value ? ' selected' : ''}`}>
      <input type="radio" name="goose-reach" aria-label={title} checked={share.reach === value} disabled={busy}
        onChange={() => send({ reach: value })} />
      <span><strong>{title}</strong><span className="small muted">{line}</span></span>
    </label>;
  return <div className="card">
    <div className="card-heading"><h2><Radio size={18} aria-hidden /> {REACH_COPY.title}</h2>
      <StatusPill status={status} /></div>
    <div className="guide-section">
      <Hint id="goose.reach" />
      <div className="goose-reach-options" role="radiogroup" aria-label={REACH_COPY.title}>
        {choice('auto', REACH_COPY.auto, REACH_COPY.autoLine)}
        {choice('local', REACH_COPY.local, REACH_COPY.localLine)}
      </div>
      {!local && adapter && <div className="goose-adapter">
        <Network size={18} aria-hidden />
        <span><span>{share.goes_out ? REACH_COPY.goesOut : REACH_COPY.listens}: <strong>{adapter}</strong></span>
          {share.goes_out && <span className="small muted">
            {share.uplink.why === 'chosen' ? REACH_COPY.chosen : REACH_COPY.picked}</span>}</span>
      </div>}
      {share.goes_out && <div className="goose-guard" aria-label={REACH_COPY.protection}>
        <span className="small muted">{REACH_COPY.protection}</span>
        <ul>{REACH_COPY.guard.map(item => <li key={item}><ShieldCheck size={14} aria-hidden />{item}</li>)}</ul>
      </div>}
      {share.message && WITH_REASON.has(share.state) && !npcapMissing &&
        <p className="small" style={{ margin: 0 }}>{share.message}</p>}
      {share.state === 'confirm' && !asking &&
        <div className="row"><button onClick={() => setAsking(true)} disabled={busy}>{REACH_COPY.review}</button></div>}
      {asking && <GooseShareWarning share={share} busy={busy} cancel={() => setAsking(false)}
        confirm={() => send({ reach: 'auto', adapter_guid: share.uplink.guid, confirm: true })} />}
      {share.alone && <p className="goose-alone" role="status"><TriangleAlert size={16} aria-hidden />
        <span>{REACH_COPY.alone(adapter)}</span></p>}
      {npcapMissing && <p className="goose-npcap" role="alert">
        <CircleAlert size={16} aria-hidden /><span>{share.message}</span>
        <a href={NPCAP_URL} target="_blank" rel="noreferrer">{REACH_COPY.getNpcap} <ExternalLink size={13} aria-hidden /></a>
        <InfoTip term="npcap" label="What is Npcap?" /></p>}
      <Disclosure title={REACH_COPY.advanced}>
        <label className="small">{REACH_COPY.adapter}{' '}
          <select value={share.uplink_guid} disabled={busy}
            onChange={event => send({ reach: share.reach, uplink_guid: event.target.value })}>
            <option value="">{REACH_COPY.automatic}</option>
            {share.choices.map(item => <option key={item.guid} value={item.guid}>
              {item.label}{item.connected ? '' : ' (not connected)'}</option>)}
          </select></label>
        {share.choices.find(item => item.guid === share.uplink_guid)?.warning &&
          <p className="small" style={{ margin: 0 }}>{share.choices.find(item => item.guid === share.uplink_guid)?.warning}</p>}
      </Disclosure>
    </div>
  </div>;
}
