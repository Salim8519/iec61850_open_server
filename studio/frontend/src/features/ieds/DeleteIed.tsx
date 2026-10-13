import { useEffect, useRef, useState } from 'react';
import { Trash2 } from 'lucide-react';
import { api } from '../../api/client';
import './import-selection.css';

type Impact = { id: string; name: string; bindings: number; sources: number;
  scenario_steps: number; scenarios_removed: number; test_controls: number };

export function DeleteIed({ iedId, disabled, perform, prepare }: {
  prepare: () => Promise<void>;
  iedId: string; disabled: boolean; perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [impact, setImpact] = useState<Impact | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (impact) dialog.current?.showModal();
    else dialog.current?.close();
  }, [impact]);
  return <>
    <button className="danger-outline" disabled={disabled || busy} onClick={async () => {
      setBusy(true); setError('');
      try { await prepare(); setImpact(await api<Impact>(`/ieds/${iedId}/removal-preview`)); }
      catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
      finally { setBusy(false); }
    }}><Trash2 size={14} />Delete IED</button>
    {!impact && error && <span role="alert" className="import-unavailable">{error}</span>}
    <dialog ref={dialog} className="import-dialog" aria-labelledby="delete-ied-title"
      onCancel={event => { event.preventDefault(); if (!busy) setImpact(null); }}>
      <h1 id="delete-ied-title">Delete {impact?.name}?</h1>
      <p className="muted">Remove this IED from the project. The original CID file is not changed.</p>
      <ul className="deletion-impact">
        <li>{impact?.bindings || 0} equipment bindings will be cleared. Drawing equipment and wires stay.</li>
        <li>{impact?.sources || 0} automatic sources will be removed.</li>
        <li>{impact?.test_controls || 0} saved Signal Test controls will be removed.</li>
        <li>{impact?.scenario_steps || 0} scenario steps will be removed;
          {' '}{impact?.scenarios_removed || 0} resulting empty sequences will also be removed.</li>
      </ul>
      <p className="small muted">A recovery project is saved under data/recovery before deletion.
        Use Open Project to restore that backup if needed.</p>
      {error && <p role="alert" className="import-unavailable">{error}</p>}
      <div className="import-actions"><span /><button disabled={busy} onClick={() => setImpact(null)}>Cancel</button>
        <button className="danger" disabled={busy || disabled} onClick={async () => {
          setBusy(true); setError('');
          await perform(async () => {
            try { await api(`/ieds/${impact?.id}`, 'DELETE', { confirm: true }); setImpact(null); }
            catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); throw reason; }
          });
          setBusy(false);
        }}>{busy ? 'Deleting…' : 'Delete IED'}</button></div>
    </dialog>
  </>;
}
