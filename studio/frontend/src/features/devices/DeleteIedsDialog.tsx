import { useEffect, useRef, useState } from 'react';
import { api } from '../../api/client';
import { SpinIcon } from '../../ui/Spinner';
import '../ieds/import-selection.css';

type Preview = { ieds: { id: string; name: string }[]; bindings: number; sources: number;
  scenario_steps: number; scenarios_removed: number; test_controls: number; fingerprint: string };

const many = (count: number, word: string) => `${count} ${word}${count === 1 ? '' : 's'}`;

/** One review for deleting one or several IEDs (bulk contract, ADR-0025 interactions):
 *  the impact of all of them together, one recovery backup, then one atomic removal. */
export function DeleteIedsDialog({ ids, close, done, perform, prepare }: {
  ids: string[]; close: () => void; done?: () => void;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    dialog.current?.showModal();
    let cancelled = false;
    prepare().then(() => api<Preview>('/ieds/removal-preview', 'POST', { ids }))
      .then(result => { if (!cancelled) setPreview(result); })
      .catch(reason => { if (!cancelled) setError(reason instanceof Error ? reason.message : String(reason)); });
    return () => { cancelled = true; };
  }, [ids.join(',')]);
  const count = preview?.ieds.length ?? ids.length;
  const title = count === 1 ? `Delete ${preview?.ieds[0]?.name ?? 'this IED'}?` : `Delete ${count} IEDs?`;
  const remove = async () => {
    if (!preview) return;
    setBusy(true); setError('');
    let removed = false;
    // perform() shows its own error line and never throws, so success is tracked here.
    await perform(async () => {
      try { await api('/ieds/remove', 'POST', { ids, fingerprint: preview.fingerprint, confirm: true }); removed = true; }
      catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); throw reason; }
    });
    setBusy(false);
    if (removed) { done?.(); close(); }
  };
  return <dialog ref={dialog} className="import-dialog" aria-labelledby="delete-ieds-title"
    onCancel={event => { event.preventDefault(); if (!busy) close(); }}>
    <h1 id="delete-ieds-title">{title}</h1>
    <p className="muted">Removes {count === 1 ? 'this IED' : 'these IEDs'} from the project. The original files are not changed.</p>
    {!preview && !error && <p className="row muted"><SpinIcon />Checking what uses {count === 1 ? 'it' : 'them'}…</p>}
    {preview && <>
      {count > 1 && <p className="small" style={{ margin: '0 0 8px', overflowWrap: 'anywhere' }}>
        {preview.ieds.map(item => item.name).join(', ')}</p>}
      <ul className="deletion-impact">
        <li>{many(preview.bindings, 'equipment connection')} will be cleared. Drawing equipment and wires stay.</li>
        <li>{many(preview.sources, 'automatic source')} will be removed.</li>
        <li>{many(preview.test_controls, 'saved test point')} will be removed.</li>
        <li>{many(preview.scenario_steps, 'scenario step')} will be removed;
          {' '}{many(preview.scenarios_removed, 'resulting empty sequence')} will also be removed.</li>
      </ul>
      <p className="small muted">A backup of the whole project is saved under data/recovery first.
        Open it with File › Open file… to get everything back.</p></>}
    {error && <p role="alert" className="import-unavailable">{error}</p>}
    <div className="import-actions"><span />
      <button disabled={busy} onClick={close}>Cancel</button>
      <button className="danger" disabled={busy || !preview} onClick={remove}>
        {busy ? <><SpinIcon />Deleting…</> : count === 1 ? 'Delete IED' : `Delete ${count} IEDs`}</button></div>
  </dialog>;
}
