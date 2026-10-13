import { useEffect, useRef, useState } from 'react';
import { api } from '../../api/client';

type Preview = { ieds: { id: string; name: string }[]; bindings: number; sources: number;
  scenario_steps: number; scenarios_removed: number; test_controls: number; fingerprint: string };
export function BulkDeleteIeds({ ids, disabled, prepare, perform, clear }: {
  ids: string[]; disabled: boolean; prepare: () => Promise<void>; clear: () => void;
  perform: (task: () => Promise<unknown>) => Promise<void>;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => { if (preview) dialog.current?.showModal(); else dialog.current?.close(); }, [preview]);
  return <><button className="danger-outline" disabled={disabled || busy || !ids.length} onClick={async () => {
    setBusy(true); setError('');
    try { await prepare(); setPreview(await api<Preview>('/ieds/removal-preview', 'POST', { ids })); }
    catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
    finally { setBusy(false); }
  }}>Delete selected ({ids.length})</button>
    {!preview && error && <span role="alert">{error}</span>}
    <dialog ref={dialog} className="import-dialog" aria-labelledby="bulk-delete-title"
      onCancel={e => { e.preventDefault(); if (!busy) setPreview(null); }}>
      <h1 id="bulk-delete-title">Delete {preview?.ieds.length} IEDs?</h1>
      <p>{preview?.ieds.map(item => item.name).join(', ')}</p>
      <ul className="deletion-impact">
        <li>{preview?.bindings} equipment bindings cleared; equipment and wires stay.</li>
        <li>{preview?.sources} automatic sources removed.</li>
        <li>{preview?.scenario_steps} scenario steps removed; {preview?.scenarios_removed} empty scenarios removed.</li>
        <li>{preview?.test_controls} saved Signal Test controls removed.</li>
      </ul>
      <p>One recovery project is saved before deletion. Original CID files are unchanged.</p>
      {error && <p role="alert">{error}</p>}
      <div className="import-actions"><span /><button disabled={busy} onClick={() => setPreview(null)}>Cancel</button>
        <button className="danger" disabled={disabled || busy} onClick={async () => {
          if (!preview) return;
          setBusy(true); setError('');
          await perform(async () => {
            try {
              await prepare();
              await api('/ieds/remove', 'POST', { ids: preview.ieds.map(item => item.id), fingerprint: preview.fingerprint, confirm: true });
              setPreview(null); clear();
            } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); throw reason; }
          });
          setBusy(false);
        }}>{busy ? 'Deleting…' : 'Confirm deletion'}</button></div>
    </dialog>
  </>;
}
