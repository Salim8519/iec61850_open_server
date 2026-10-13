import { useEffect, useRef, useState } from 'react';
import './import-selection.css';

export type ImportPreview = {
  file: File; available_slots: number;
  ieds: { name: string; description: string; manufacturer: string; type: string;
    logical_devices: number; unavailable_reason: string }[];
};

export function ImportSelection({ preview, busy, error, confirm, cancel }: {
  preview: ImportPreview; busy: boolean; error: string;
  confirm: (names: string[]) => void; cancel: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [selected, setSelected] = useState<string[]>(() => {
    const usable = preview.ieds.filter(ied => !ied.unavailable_reason);
    return usable.length === 1 ? [usable[0].name] : [];
  });
  const [query, setQuery] = useState('');
  useEffect(() => {
    const element = dialog.current;
    element?.showModal();
    return () => element?.close();
  }, []);
  const visible = preview.ieds.filter(ied =>
    `${ied.name} ${ied.description} ${ied.manufacturer} ${ied.type}`.toLowerCase().includes(query.toLowerCase()));
  return <dialog ref={dialog} className="import-dialog" aria-labelledby="import-title"
    onCancel={event => { event.preventDefault(); if (!busy) cancel(); }}>
    <h1 id="import-title">Choose IEDs to add</h1>
    <p className="muted small">{preview.file.name} · {preview.ieds.length} IEDs found.
      Nothing is added until you confirm.</p>
    <input aria-label="Search IEDs in import" placeholder="Search name, manufacturer or description"
      value={query} onChange={event => setQuery(event.target.value)} disabled={busy} />
    <div className="import-list">
      {visible.map(ied => <label className="import-choice" key={ied.name}>
        <input type="checkbox" checked={selected.includes(ied.name)} aria-label={`Add ${ied.name}`}
          disabled={busy || !!ied.unavailable_reason ||
            (!selected.includes(ied.name) && selected.length >= preview.available_slots)}
          onChange={event => setSelected(names => event.target.checked ? [...names, ied.name] : names.filter(name => name !== ied.name))} />
        <span><strong>{ied.name}</strong><small>{[ied.manufacturer, ied.type].filter(Boolean).join(' · ') || 'Manufacturer not specified'}
          {' · '}{ied.logical_devices} logical devices</small>
          {ied.description && <small>{ied.description}</small>}
          {ied.unavailable_reason && <small className="import-unavailable">{ied.unavailable_reason}</small>}
        </span>
      </label>)}
      {!visible.length && <p className="muted">No matching IEDs.</p>}
    </div>
    {error && <p className="import-unavailable" role="alert">{error}</p>}
    <div className="import-actions"><span className="muted small">{selected.length} selected · {preview.available_slots} slots available</span>
      <button onClick={cancel} disabled={busy}>Cancel</button>
      <button className="primary" onClick={() => confirm(selected)} disabled={busy || !selected.length}>
        {busy ? 'Importing…' : `Add ${selected.length || ''} selected IED${selected.length === 1 ? '' : 's'}`}
      </button>
    </div>
  </dialog>;
}
