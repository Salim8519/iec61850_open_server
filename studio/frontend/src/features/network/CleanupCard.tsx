import { Eraser } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { PlanItem } from '../../api/network';
import { Hint } from '../../ui/Hint';
import { StatusPill } from '../../ui/StatusPill';
import { UI_STATUS } from '../../copy/status';

type CleanupPreview = { items: PlanItem[]; fingerprint: string };

export function CleanupCard({ locked, revision, perform }: {
  locked: boolean; revision: number; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const [preview, setPreview] = useState<CleanupPreview | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [message, setMessage] = useState('');
  useEffect(() => {
    api<CleanupPreview>('/network/cleanup/preview').then(setPreview).catch(() => setPreview(null));
  }, [revision]);
  if (!preview?.items.length && !message) return null;
  return <div className="card">
    <div className="card-heading"><h2>ScadaBench network changes</h2></div>
    <div className="guide-section">
      {message ? <p style={{ margin: 0 }}>{message}</p> : <>
        <Hint id={confirming ? 'network.cleanup' : 'network.leftovers'} />
        <table><tbody>{preview!.items.map((item, index) => <tr key={index}><td>{item.description}</td>
          <td><StatusPill status={UI_STATUS.item_remove} /></td></tr>)}</tbody></table>
        <div className="row">
          {!confirming && <button disabled={locked} onClick={() => setConfirming(true)}><Eraser size={14} />Clean up…</button>}
          {confirming && <><button onClick={() => setConfirming(false)}>Keep them</button>
            <button className="danger" disabled={locked} onClick={() => perform(async () => {
              const result = await api<{ message: string }>('/network/cleanup/apply', 'POST',
                { fingerprint: preview!.fingerprint, confirm: true });
              setMessage(result.message); setConfirming(false);
            })}><Eraser size={14} />Remove these changes</button></>}
          {locked && <Hint id="network.locked" compact />}
        </div></>}
    </div>
  </div>;
}
