import { Radio } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { GooseOverview } from '../../api/network';
import { EmptyState } from '../../ui/EmptyState';
import { Hint } from '../../ui/Hint';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import { gooseStatus, UI_STATUS } from '../../copy/status';
import { shortBlock } from './endpoints';

export function GooseTraffic({ running, revision, lab }: { running: boolean; revision: number; lab: boolean }) {
  const [overview, setOverview] = useState<GooseOverview | null>(null);
  const [error, setError] = useState('');
  useEffect(() => {
    let cancelled = false;
    const load = () => api<GooseOverview>('/goose').then(value => { if (!cancelled) { setOverview(value); setError(''); } })
      .catch(reason => { if (!cancelled) setError(reason.message); });
    load();
    const timer = running ? setInterval(load, 2000) : undefined;
    return () => { cancelled = true; if (timer) clearInterval(timer); };
  }, [running, revision]);
  const npcap = overview?.npcap;
  return <div className="card">
    <div className="card-heading"><h2><InfoTip term="goose" label="GOOSE traffic" /></h2>
      {npcap && <StatusPill status={npcap.installed ? UI_STATUS.npcap_ready : UI_STATUS.npcap_missing} />}</div>
    <div className="guide-section">
      <Hint id={lab ? 'goose.intro' : 'goose.loopback'} />
      {npcap && !npcap.installed && <div className="row"><Hint id="goose.npcap.missing" />
        <a className="button" href={npcap.download} target="_blank" rel="noreferrer">Get Npcap</a></div>}
      {error && <p role="alert" style={{ color: 'var(--error)', margin: 0 }}>{error}</p>}
    </div>
    {overview && !overview.watches.length ? <EmptyState icon={<Radio size={20} />} title="No GOOSE in this project"
      hintId="goose.intro" /> : <table>
      <thead><tr><th><InfoTip term="gocb" label="GOOSE block" /></th><th>From</th><th>To</th>
        <th><InfoTip term="appid" label="APPID" /> / <InfoTip term="mac" label="MAC" /></th><th>Status</th>
        <th><InfoTip term="stnum" label="Messages" /></th></tr></thead>
      <tbody>{(overview?.watches || []).map(watch => <tr key={watch.key}>
        <td><strong>{shortBlock(watch.gocb_ref)}</strong><div className="small muted"><code>{watch.gocb_ref}</code></div></td>
        <td>{watch.publisher?.name || 'Unknown'}<div className="small muted">{watch.publisher?.role === 'real' ? 'real device'
          : watch.publisher?.role === 'outside_project' ? 'not in project' : 'simulated'}</div></td>
        <td>{watch.subscribers.length ? watch.subscribers.map(item => item.name).join(', ') : <span className="muted">Nobody in project</span>}</td>
        <td><code>{`0x${watch.app_id.toString(16).toUpperCase().padStart(4, '0')}`}</code><div className="small"><code>{watch.mac}</code></div></td>
        <td>{lab ? <StatusPill status={gooseStatus(watch.state, watch.problem)} showHint={watch.state !== 'receiving'} />
          : <StatusPill status={UI_STATUS.goose_needs_lab} />}</td>
        <td className="small">{watch.frames ? `${watch.frames} seen · state ${watch.st_num ?? '-'}` : '–'}</td>
      </tr>)}</tbody></table>}
    {!!overview?.unknown.length && <div className="guide-section"><Hint id="goose.unknown" />
      <ul className="small">{overview.unknown.map(item => <li key={item.gocb_ref}><code>{item.gocb_ref}</code> · APPID {item.app_id} · {item.mac}</li>)}</ul></div>}
  </div>;
}
