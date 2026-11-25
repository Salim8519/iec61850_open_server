import { useEffect, useRef, useState } from 'react';
import { api } from '../../api/client';
import { REIMPORT_COPY } from '../../copy/reimport';
import { SpinIcon } from '../../ui/Spinner';
import '../ieds/import-selection.css';

type Counts = { before: number; after: number; added: number; removed: number };
type Answer = { name: string; filename: string; same_file: boolean; signals: Counts; controls: Counts; reports: Counts;
  kept: number; lost: { where: string; ref: string; why: string }[]; gateway_gone: string[]; applied: boolean };

const send = (iedId: string, file: File, apply: boolean) => {
  const form = new FormData();
  form.append('file', file);
  form.append('apply', apply ? '1' : '0');
  return api<Answer>(`/ieds/${encodeURIComponent(iedId)}/reimport`, 'POST', form);
};

/** Re-import a newer CID of one IED (ADR-0150): pick the file, read what changes (the engine
 *  works it out), then apply. The IED keeps its id, settings and every link the file still has. */
export function ReimportIedDialog({ iedId, iedName, close, perform, prepare }: {
  iedId: string; iedName: string; close: () => void;
  perform: (task: () => Promise<unknown>) => Promise<void>; prepare: () => Promise<void>;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<Answer | null>(null);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => { dialog.current?.showModal(); }, []);
  useEffect(() => {
    if (!file) return;
    let cancelled = false;
    setPreview(null); setError(''); setBusy(true);
    prepare().then(() => send(iedId, file, false))
      .then(answer => { if (!cancelled) setPreview(answer); })
      .catch(reason => { if (!cancelled) setError(reason instanceof Error ? reason.message : String(reason)); })
      .finally(() => { if (!cancelled) setBusy(false); });
    return () => { cancelled = true; };
  }, [file]);
  const apply = async () => {
    if (!file || !preview) return;
    setBusy(true); setError('');
    let applied = false;
    await perform(async () => {
      try { applied = (await send(iedId, file, true)).applied; }
      catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); throw reason; }
    });
    setBusy(false);
    if (applied) close();
  };
  return <dialog ref={dialog} className="import-dialog" aria-labelledby="reimport-title"
    onCancel={event => { event.preventDefault(); if (!busy) close(); }}>
    <h1 id="reimport-title">{REIMPORT_COPY.title(iedName)}</h1>
    <p className="muted">{REIMPORT_COPY.intro}</p>
    <label className="button reimport-pick">
      <input type="file" accept=".cid,.icd,.iid,.scd,.xml" hidden disabled={busy}
        onChange={event => setFile(event.target.files?.[0] ?? null)} />
      {file ? REIMPORT_COPY.another : REIMPORT_COPY.choose}</label>
    {file && <p className="small" style={{ margin: 0, overflowWrap: 'anywhere' }}>{file.name}</p>}
    {busy && !preview && <p className="row muted"><SpinIcon />{REIMPORT_COPY.checking}</p>}
    {preview && <ReimportSummary answer={preview} />}
    {error && <p role="alert" className="import-unavailable">{error}</p>}
    <div className="import-actions"><span />
      <button disabled={busy} onClick={close}>Cancel</button>
      <button className={preview?.lost.length ? 'danger' : 'primary'} disabled={busy || !preview || preview.same_file}
        onClick={() => { void apply(); }}>
        {busy && preview ? <><SpinIcon />{REIMPORT_COPY.applying}</> : REIMPORT_COPY.apply(preview?.lost.length ?? 0)}</button></div>
  </dialog>;
}

function ReimportSummary({ answer }: { answer: Answer }) {
  if (answer.same_file) return <p className="small" role="status">{REIMPORT_COPY.same}</p>;
  return <div className="stack" style={{ gap: 8 }}>
    <ul className="deletion-impact">
      <li>{REIMPORT_COPY.counts('Data points', answer.signals)}</li>
      <li>{REIMPORT_COPY.counts('Commands', answer.controls)}</li>
      <li>{REIMPORT_COPY.counts('Reports', answer.reports)}</li>
      <li>{REIMPORT_COPY.kept(answer.kept)}</li>
    </ul>
    {answer.lost.length > 0 ? <>
      <strong>{REIMPORT_COPY.lostTitle(answer.lost.length)}</strong>
      <ul className="reimport-lost">{answer.lost.map((item, index) =>
        <li key={index} className="small"><span>{item.where}</span>{item.ref && <> · <code>{item.ref}</code></>}
          <span className="muted"> — {item.why}</span></li>)}</ul></>
      : <p className="small" role="status">{REIMPORT_COPY.nothingLost}</p>}
    {answer.gateway_gone.length > 0 && <p className="small">{REIMPORT_COPY.gateway(answer.gateway_gone)}</p>}
    <p className="small muted">{REIMPORT_COPY.backup}</p>
  </div>;
}
