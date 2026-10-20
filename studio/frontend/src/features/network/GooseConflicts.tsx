import { TriangleAlert } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { GooseConflict, GooseShare } from '../../api/network';
import type { Project } from '../../api/types';
import { useEventCount } from '../../state/useLive';
import { CONFLICT_COPY } from '../../copy/gooseReach';
import './goose.css';

/** Asks once per conflict: a real device sends the same GOOSE as a simulated IED (GOOSE-21).
 * The engine already holds that IED's outward copy; the answer lasts for the current Run. */
export function GooseConflicts({ project }: { project: Project }) {
  const [open, setOpen] = useState<GooseConflict[]>([]);
  const [sure, setSure] = useState('');
  const latest = useEventCount('goose_conflict');
  const running = project.runtime !== 'stopped';
  useEffect(() => {
    if (!running) { setOpen([]); return; }
    if (!latest) return;
    api<GooseShare>('/goose/share').then(share => setOpen(share.conflicts.filter(item => !item.choice)))
      .catch(() => {});
  }, [latest, running]);
  if (!open.length) return null;
  const name = (id: string) => project.ieds.find(ied => ied.id === id)?.name || id;
  const answer = (conflict: GooseConflict, choice: 'stop' | 'continue') =>
    api('/goose/conflicts', 'POST', { ied_id: conflict.ied_id, choice, confirm: choice === 'continue' })
      .then(() => { setOpen(items => items.filter(item => item.ied_id !== conflict.ied_id)); setSure(''); })
      .catch(() => {});
  return <div className="goose-conflicts">{open.map(conflict =>
    <div key={conflict.ied_id} className="goose-confirm real" role="alertdialog" aria-label={CONFLICT_COPY.title(name(conflict.ied_id))}>
      <p><TriangleAlert size={18} aria-hidden /><strong>{CONFLICT_COPY.title(name(conflict.ied_id))}</strong></p>
      <p className="small">{CONFLICT_COPY.body(conflict.src_mac, conflict.gocb_ref)}</p>
      {sure === conflict.ied_id && <p className="small"><strong>{CONFLICT_COPY.sure}</strong></p>}
      <div className="row">
        <button autoFocus onClick={() => answer(conflict, 'stop')}>{CONFLICT_COPY.stop}</button>
        {sure === conflict.ied_id
          ? <button className="danger" onClick={() => answer(conflict, 'continue')}>{CONFLICT_COPY.yes}</button>
          : <button onClick={() => setSure(conflict.ied_id)}>{CONFLICT_COPY.continue}</button>}
      </div>
    </div>)}</div>;
}
