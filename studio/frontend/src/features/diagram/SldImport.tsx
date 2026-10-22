import { useEffect, useRef, useState } from 'react';
import { Network } from 'lucide-react';
import { api } from '../../api/client';
import type { Diagram } from '../../api/types';
import { Disclosure } from '../../ui/Disclosure';
import { Hint } from '../../ui/Hint';
import { SpinIcon } from '../../ui/Spinner';
import { SegmentedControl } from '../../ui/SegmentedControl';
import { previewIsCurrent, sldAvailability, sldSummary, type SldMode, type SldPreview, type SldSource } from './sldPreview';
import '../ieds/import-selection.css';

/** "Build from SCD": preview the single-line diagram from the SCD, then apply it as one undo step. */
export function SldImport({ disabled, revision, apply, prepare }: {
  disabled: boolean; revision: number; apply: (diagram: Diagram) => void; prepare: () => Promise<void>;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [sources, setSources] = useState<SldSource[] | null>(null);
  const [source, setSource] = useState('');
  const [mode, setMode] = useState<SldMode>('beside');
  const [preview, setPreview] = useState<SldPreview | null>(null);
  const [confirmed, setConfirmed] = useState(false);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => { if (sources) dialog.current?.showModal(); else dialog.current?.close(); }, [sources]);
  const run = async (task: () => Promise<void>) => {
    setBusy(true); setError('');
    try { await task(); } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
    finally { setBusy(false); }
  };
  const open = () => run(async () => {
    const result = await api<{ sources: SldSource[] }>('/project/substation-sources');
    const usable = sldAvailability(result.sources).usable;
    setSource(usable[0]?.id ?? ''); setPreview(null); setConfirmed(false); setSources(result.sources);
  });
  const makePreview = (nextMode = mode, nextSource = source) => run(async () => {
    setPreview(null); setConfirmed(false);
    await prepare();
    setPreview(await api<SldPreview>('/project/diagram/from-scl/preview', 'POST', { source_id: nextSource, mode: nextMode }));
  });
  const availability = sldAvailability(sources ?? []);
  const close = () => { setSources(null); setPreview(null); };
  const stale = preview && !previewIsCurrent(preview, revision);
  return <>
    <button className="ghost" data-tour="diagram-build" disabled={disabled || busy} onClick={open}
      title="Draw the single-line diagram from the substation SCD file">{busy && !sources ? <SpinIcon size={17} />
        : <Network size={17} />}{busy && !sources ? 'Reading the SCD…' : 'Build from SCD'}</button>
    {!sources && error && <span role="alert" className="import-unavailable">{error}</span>}
    <dialog ref={dialog} className="import-dialog" aria-labelledby="sld-import-title"
      onCancel={event => { event.preventDefault(); if (!busy) close(); }}>
      <h1 id="sld-import-title">Build the diagram from the SCD</h1>
      <Hint id="sld.intro" />
      {!availability.enabled ? <p className="import-unavailable">{availability.hint}</p> : <>
        {availability.usable.length > 1 && <label>File
          <select value={source} disabled={busy} onChange={event => { setSource(event.target.value); setPreview(null); }}>
            {availability.usable.map(item => <option key={item.id} value={item.id}>
              {item.filename} · {item.substations.join(', ')} · {item.equipment} items</option>)}
          </select></label>}
        <SegmentedControl label="Where to draw" value={mode} disabled={busy} full
          options={[{ value: 'beside', label: 'Add beside my drawing' }, { value: 'replace', label: 'Replace my drawing' }]}
          onChange={value => { setMode(value); setPreview(null); }} />
        <Hint id={mode === 'replace' ? 'sld.replace' : 'sld.beside'} compact />
        {preview && <div className="sheet-section">
          <ul className="deletion-impact">{sldSummary(preview).map(line => <li key={line}>{line}</li>)}</ul>
          {preview.unbound.length > 0 && <Disclosure title={`Needs a link (${preview.unbound.length})`}>
            <Hint id="sld.unbound" compact />
            <ul className="deletion-impact">{preview.unbound.map(item => <li key={item.path}>
              <strong>{item.path}</strong> ({item.kind}): {[...new Set(item.reasons)].join('; ')}</li>)}</ul></Disclosure>}
          {preview.skipped.length > 0 && <Disclosure title={`No symbol yet (${preview.skipped.length})`}>
            <ul className="deletion-impact">{preview.skipped.map(item => <li key={item.path}>
              <strong>{item.path}</strong> ({item.type}): {item.reason}</li>)}</ul></Disclosure>}
          {preview.diagnostics.length > 0 && <Disclosure title={`Notes about the file (${preview.diagnostics.length})`}>
            <ul className="deletion-impact">{preview.diagnostics.map(line => <li key={line}>{line}</li>)}</ul></Disclosure>}
          {mode === 'replace' && preview.removed > 0 && <label className="import-choice">
            <input type="checkbox" checked={confirmed} onChange={event => setConfirmed(event.target.checked)} />
            <span>Yes, remove the {preview.removed} items on my diagram. Undo brings them back.</span></label>}
          {stale && <p role="alert" className="import-unavailable">The project changed after this preview. Press Preview again.</p>}
        </div>}
      </>}
      {error && <p role="alert" className="import-unavailable">{error}</p>}
      <div className="import-actions"><span />
        <button disabled={busy} onClick={close}>Cancel</button>
        {availability.enabled && <button disabled={busy || !source} onClick={() => makePreview()}>
          {busy ? 'Working…' : preview ? 'Preview again' : 'Preview'}</button>}
        {preview && <button className="primary" disabled={busy || disabled || !!stale
          || (mode === 'replace' && preview.removed > 0 && !confirmed)}
          onClick={() => { apply(preview.diagram); close(); }}>Apply to diagram</button>}
      </div>
    </dialog>
  </>;
}
