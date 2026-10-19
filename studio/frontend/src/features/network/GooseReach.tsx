import { Radio } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { GooseShare } from '../../api/network';
import { REACH_COPY, REACH_STATES, reachStatus } from '../../copy/gooseReach';
import { Disclosure } from '../../ui/Disclosure';
import { Hint } from '../../ui/Hint';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import { GooseShareWarning } from './GooseShareWarning';
import './goose.css';

const NPCAP_URL = 'https://npcap.com/#download';

/** "Where GOOSE goes": inside ScadaBench (default) or also shared with the network (ADR-0073).
 * Allowed while running; the backend applies it to the live GOOSE engine at once. */
export function GooseReach({ revision, perform }: { revision: number; perform: (task: () => Promise<unknown>) => Promise<void> }) {
  const [share, setShare] = useState<GooseShare | null>(null);
  const [asking, setAsking] = useState(false);
  const [busy, setBusy] = useState(false);
  const load = useCallback(() => api<GooseShare>('/goose/share').then(setShare).catch(() => {}), []);
  useEffect(() => { load(); }, [load, revision]);
  useEffect(() => {
    if (!share?.running) return undefined;
    const timer = setInterval(load, 2000);
    return () => clearInterval(timer);
  }, [share?.running, load]);
  if (!share) return null;
  const send = (body: Record<string, unknown>) => {
    setBusy(true);
    return perform(() => api<GooseShare>('/goose/share', 'PUT', body).then(setShare))
      .finally(() => { setBusy(false); setAsking(false); });
  };
  const outside = share.reach === 'outside';
  const adapter = share.uplink.guid ? share.uplink.label : '';
  const canShare = share.npcap && !!share.uplink.guid;
  const status = !share.running && outside ? REACH_STATES.share_at_run : reachStatus(share.state);
  return <div className="card">
    <div className="card-heading"><h2><Radio size={18} aria-hidden /> {REACH_COPY.title}</h2>
      <StatusPill status={status} /></div>
    <div className="guide-section">
      <Hint id="goose.reach" />
      <div className="goose-reach-options" role="radiogroup" aria-label={REACH_COPY.title}>
        <label className={`goose-reach-option${outside ? '' : ' selected'}`}>
          <input type="radio" name="goose-reach" aria-label={REACH_COPY.inside} checked={!outside} disabled={busy}
            onChange={() => send({ reach: 'inside' })} />
          <span><strong>{REACH_COPY.inside}</strong><span className="small muted">{REACH_COPY.insideLine}</span></span>
        </label>
        <label className={`goose-reach-option${outside ? ' selected' : ''}`}>
          <input type="radio" name="goose-reach" aria-label={REACH_COPY.outside} checked={outside} disabled={busy || (!outside && !canShare)}
            onChange={() => setAsking(true)} />
          <span><strong>{REACH_COPY.outside}</strong><span className="small muted">{REACH_COPY.outsideLine}</span></span>
        </label>
      </div>
      {asking && <GooseShareWarning share={share} busy={busy} cancel={() => setAsking(false)}
        confirm={() => send({ reach: 'outside', adapter_guid: share.uplink.guid, confirm: true })} />}
      {share.message && share.state !== 'inside' && <p className="small" style={{ margin: 0 }}>{share.message}</p>}
      <p className="small" style={{ margin: 0 }}>{REACH_COPY.adapter}: <strong>{adapter || '—'}</strong>
        {adapter && share.uplink.auto && <span className="muted"> · {REACH_COPY.auto}</span>}
        {!adapter && <span className="muted"> · {share.uplink.reason}</span>}</p>
      {share.npcap ? adapter && <Hint id="goose.listen" compact />
        : <div className="row"><Hint id="goose.npcap.missing" /><InfoTip term="npcap" label="What is Npcap?" />
          <a className="button" href={NPCAP_URL} target="_blank" rel="noreferrer">{REACH_COPY.getNpcap}</a></div>}
      <Disclosure title={REACH_COPY.advanced}>
        <label className="small">{REACH_COPY.adapter}{' '}
          <select value={share.uplink_guid} disabled={busy || outside}
            onChange={event => send({ reach: 'inside', uplink_guid: event.target.value })}>
            <option value="">{REACH_COPY.automatic}</option>
            {share.choices.map(choice => <option key={choice.guid} value={choice.guid}>
              {choice.label}{choice.connected ? '' : ' (not connected)'}</option>)}
          </select></label>
        {outside && <p className="small muted" style={{ margin: 0 }}>{REACH_COPY.lockedWhileSharing}</p>}
        {share.choices.find(choice => choice.guid === share.uplink_guid)?.warning &&
          <p className="small" style={{ margin: 0 }}>{share.choices.find(choice => choice.guid === share.uplink_guid)?.warning}</p>}
      </Disclosure>
    </div>
  </div>;
}
