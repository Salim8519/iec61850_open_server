import { useEffect, useRef, useState } from 'react';
import { Save } from 'lucide-react';
import type { Project } from '../../api/types';
import { PROJECT_FILE_COPY as COPY } from '../../copy/projectFile';
import { saveProjectAs } from '../../state/saveProjectAs';
import { SpinIcon } from '../../ui/Spinner';
import '../ieds/import-selection.css';

export type ProjectFileMode = 'new' | 'close';

/** File › New project and File › Close project (ADR-0135). One project is open at a time, so
 *  both close the open one; the dialog offers to save a copy first. New asks for a name. */
export function ProjectFileDialog({ mode, project, close, confirm, prepare }: {
  mode: ProjectFileMode; project: Project; close: () => void; confirm: (name: string) => Promise<boolean>;
  prepare: () => Promise<void>;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState('');
  const [error, setError] = useState('');
  useEffect(() => { dialog.current?.showModal(); }, []);
  const hasWork = project.ieds.length > 0 || project.diagram.nodes.length > 0;
  const copy = async () => {
    setError(''); setNote('');
    try {
      await prepare();  // a diagram edit still waiting to be saved goes into the copy
      const result = await saveProjectAs(project.name);
      setNote(result === 'cancelled' ? '' : result === 'saved' ? COPY.copySaved : COPY.copyDownloaded);
    } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
  };
  const go = async () => {
    setBusy(true); setError('');
    const done = await confirm(mode === 'new' ? name.trim() : '');
    setBusy(false);
    if (done) close();
  };
  return <dialog ref={dialog} className="import-dialog project-file-dialog" aria-labelledby="project-file-title"
    onCancel={event => { event.preventDefault(); if (!busy) close(); }}>
    <h1 id="project-file-title">{mode === 'new' ? COPY.newTitle : COPY.closeTitle(project.name)}</h1>
    <p className="muted">{mode === 'new' ? COPY.newIntro(project.name) : COPY.closeIntro}</p>
    {mode === 'new' && <label className="project-name-field">{COPY.nameLabel}
      <input value={name} maxLength={80} placeholder={COPY.namePlaceholder} autoFocus disabled={busy}
        onChange={event => setName(event.target.value)}
        onKeyDown={event => { if (event.key === 'Enter' && !busy) { event.preventDefault(); go(); } }} /></label>}
    {hasWork && <p className="small">{COPY.copyHint}</p>}
    {note && <p className="small" role="status">✓ {note}</p>}
    {error && <p role="alert" className="import-unavailable">{error}</p>}
    <div className="import-actions">
      {hasWork && <button disabled={busy} onClick={copy}><Save size={16} />{COPY.saveCopy}</button>}<span />
      <button disabled={busy} onClick={close}>{COPY.cancel}</button>
      <button className="primary" disabled={busy} onClick={go}>
        {busy ? <><SpinIcon />{COPY.working}</> : mode === 'new' ? COPY.newAction : COPY.closeAction}</button></div>
  </dialog>;
}
