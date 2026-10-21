import { Radio } from 'lucide-react';
import { useLiveFeed } from '../../api/useLiveFeed';
import type { GooseOverview } from '../../api/network';
import { EmptyState } from '../../ui/EmptyState';
import { Hint } from '../../ui/Hint';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import { gooseStatus } from '../../copy/status';
import { reachStatus } from '../../copy/gooseReach';
import { shortBlock } from './endpoints';

/** Every GOOSE block of the project and what really arrives; GOOSE runs in every network mode (ADR-0073). */
export function GooseTraffic({ running, revision }: { running: boolean; revision: number }) {
  const { value: overview, error } = useLiveFeed<GooseOverview>('/goose', running ? 2000 : null, revision);
  return <div className="card">
    <div className="card-heading"><h2><InfoTip term="goose" label="GOOSE traffic" /></h2>
      {overview && <StatusPill status={reachStatus(overview.state)} />}</div>
    <div className="guide-section">
      <Hint id="goose.intro" />
      {overview?.message && <p className="small" style={{ margin: 0 }}>{overview.message}</p>}
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
        <td><StatusPill status={gooseStatus(watch.state, watch.problem)} showHint={watch.state !== 'receiving'} />
          {watch.origin === 'wire' && <div className="small muted">from the network{watch.src_mac ? ` · ${watch.src_mac}` : ''}</div>}</td>
        <td className="small">{watch.frames ? `${watch.frames} seen · state ${watch.st_num ?? '-'}` : '–'}</td>
      </tr>)}</tbody></table>}
    {!!overview?.unknown.length && <div className="guide-section"><Hint id="goose.unknown" />
      <ul className="small">{overview.unknown.map(item => <li key={item.gocb_ref}><code>{item.gocb_ref}</code> · APPID {item.app_id} · {item.mac}</li>)}</ul></div>}
  </div>;
}
