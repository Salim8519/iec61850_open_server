import { useEffect, useRef, useState } from 'react';
import { Download, X } from 'lucide-react';
import { api } from '../../api/client';
import { download } from '../../api/download';
import type { Ied, Signal } from '../../api/types';
import './import-selection.css';

export function ExportScl({ ied }: { ied: Ied }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [signals, setSignals] = useState<Signal[]>([]);
  const [selected, setSelected] = useState<Signal | null>(null);
  const [draft, setDraft] = useState('');
  const [overrides, setOverrides] = useState<Record<string, Signal['value']>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => { if (open) dialog.current?.showModal(); else dialog.current?.close(); }, [open]);
  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    const timer = setTimeout(() => {
      api<{ items: Signal[] }>(`/ieds/${ied.id}/signals?q=${encodeURIComponent(query)}&limit=100`)
        .then(result => { if (!cancelled) setSignals(result.items); })
        .catch(reason => { if (!cancelled) setError(reason.message); });
    }, 150);
    return () => { cancelled = true; clearTimeout(timer); };
  }, [ied.id, open, query]);
  const supported = (signal: Signal) => !signal.ref.includes('(') && (
    signal.kind.startsWith('INT') || signal.kind.startsWith('FLOAT') || signal.kind.includes('String')
    || ['BOOLEAN', 'Enum', 'Timestamp', 'Currency', 'ObjRef'].includes(signal.kind));
  function stage() {
    if (!selected) return;
    const numeric = selected.kind.startsWith('INT') || selected.kind.startsWith('FLOAT')
      || ['Enum', 'Timestamp'].includes(selected.kind);
    const value = selected.kind === 'BOOLEAN' ? draft === 'true' : numeric ? Number(draft) : draft;
    if (numeric && (!draft.trim() || !Number.isFinite(value))) {
      setError('Enter a finite numeric value.'); return;
    }
    if (numeric && !selected.kind.startsWith('FLOAT') && !Number.isSafeInteger(value)) {
      setError('Enter an integer within the browser-safe numeric range.'); return;
    }
    setOverrides(previous => ({ ...previous, [selected.ref]: value }));
    setError(''); setSelected(null);
  }
  return <>
    <button disabled={ied.validation?.status !== 'valid'}
      title="Requires a supported, schema-valid source document" onClick={() => {
        setError(''); setOverrides({}); setSelected(null); setOpen(true);
      }}><Download size={14} />Export SCL</button>
    <dialog ref={dialog} className="import-dialog" aria-labelledby="export-scl-title"
      onCancel={event => { event.preventDefault(); if (!busy) setOpen(false); }}>
      <h1 id="export-scl-title">Export SCL — {ied.name}</h1>
      <p className="muted">Download a verified copy of the original document, optionally changing
        selected initial values. Your active project and original CID stay unchanged.</p>
      <p className="small">All IED declarations in the original file are retained, including unselected
        GOOSE peers. Original IP addresses stay. Canvas equipment and wires are not exported yet.</p>
      <fieldset disabled={busy}>
        <legend>Optional initial-value edits</legend>
        <label>Find a data reference<input value={query} onChange={event => setQuery(event.target.value)}
          placeholder="Search exact reference or logical node" /></label>
        <label>Signal (first 100 search results)<select value={selected?.ref || ''} onChange={event => {
          const signal = signals.find(item => item.ref === event.target.value) || null;
          setSelected(signal); setDraft(signal ? String(overrides[signal.ref] ?? signal.value) : '');
        }}><option value="">Choose a signal to edit</option>
          {signals.map(signal => <option key={signal.ref} value={signal.ref} disabled={!supported(signal)}>
            {signal.ref} ({signal.kind}){supported(signal) ? '' : ' — encoding not supported yet'}
          </option>)}
        </select></label>
        {selected && <div className="row" style={{ alignItems: 'end', marginTop: 12 }}>
          <label style={{ flex: 1 }}>New initial value{selected.kind === 'Timestamp' ? ' (Unix milliseconds)' : ''}
            {selected.kind === 'BOOLEAN' ? <select value={draft} onChange={event => setDraft(event.target.value)}>
              <option value="false">False</option><option value="true">True</option></select>
              : selected.kind === 'Enum' ? <select value={draft} onChange={event => setDraft(event.target.value)}>
                {Object.entries(selected.enums).map(([label, value]) => <option key={label} value={value}>{label} ({value})</option>)}
              </select> : <input value={draft} onChange={event => setDraft(event.target.value)} />}
          </label><button onClick={stage}>Stage edit</button></div>}
      </fieldset>
      <h2 style={{ marginTop: 16 }}>{Object.keys(overrides).length} staged edits</h2>
      <div style={{ maxHeight: 160, overflow: 'auto' }}>
        {Object.entries(overrides).map(([ref, value]) => <div className="row" key={ref}>
          <code style={{ flex: 1, overflowWrap: 'anywhere' }}>{ref} = {String(value)}</code>
          <button disabled={busy} aria-label={`Remove export edit ${ref}`} onClick={() => setOverrides(previous => {
            const next = { ...previous }; delete next[ref]; return next;
          })}><X size={14} /></button></div>)}
      </div>
      {error && <p role="alert" className="import-unavailable">{error}</p>}
      <p className="small muted">Download runs XSD validation and signal round-trip checks.
        Unsupported edits return a diagnostic instead of a file.</p>
      <div className="import-actions"><span /><button disabled={busy} onClick={() => setOpen(false)}>Cancel</button>
        <button className="primary" disabled={busy || selected !== null} onClick={async () => {
          setBusy(true); setError('');
          try {
            await download(`/ieds/${ied.id}/export-scl`, { overrides }, `${ied.name}.edited.scd`);
            setOpen(false);
          } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
          finally { setBusy(false); }
        }}>{busy ? 'Validating export…' : 'Validate & download'}</button></div>
    </dialog>
  </>;
}
