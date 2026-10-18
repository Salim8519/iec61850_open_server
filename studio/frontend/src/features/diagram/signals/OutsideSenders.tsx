import { useEffect, useRef, useState } from 'react';
import { ChevronDown, FileUp } from 'lucide-react';
import { OUTSIDE_COPY } from '../../../copy/signalLayers';
import type { SignalLayerState } from '../../../equipment/signalLayerStore';

/** Toolbar control next to the GOOSE switch: "N senders outside project ▾" opens a small list.
 * Senders outside the project never get a tag on the canvas (change 0097, SLD stays clean). */
export function OutsideSenders({ state, importFile }: { state: SignalLayerState; importFile: () => void }) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLSpanElement>(null);
  useEffect(() => {
    if (!open) return;
    const close = (event: MouseEvent | KeyboardEvent) => {
      if (event instanceof KeyboardEvent ? event.key === 'Escape' : !root.current?.contains(event.target as Node)) setOpen(false);
    };
    window.addEventListener('mousedown', close);
    window.addEventListener('keydown', close);
    return () => { window.removeEventListener('mousedown', close); window.removeEventListener('keydown', close); };
  }, [open]);
  const receivers = new Set(state.map.ieds.filter(ied => ied.in_project).map(ied => ied.name));
  const senders = state.map.ieds.filter(ied => !ied.in_project
    && state.map.links.some(link => link.publisher === ied.name && receivers.has(link.subscriber)));
  if (!state.layers.goose || !senders.length) return null;
  return <span ref={root} className="outside-senders">
    <button className="ghost" aria-expanded={open} onClick={() => setOpen(!open)} title={OUTSIDE_COPY.intro}>
      {OUTSIDE_COPY.button(senders.length)}<ChevronDown size={15} aria-hidden /></button>
    {open && <div className="outside-popover" role="dialog" aria-label={OUTSIDE_COPY.button(senders.length)}>
      <p className="small muted">{OUTSIDE_COPY.intro}</p>
      {senders.map(ied => <div key={ied.name} className="outside-row">
        <strong title={[ied.desc, ied.manufacturer].filter(Boolean).join(' · ') || undefined}>{ied.name}</strong>
        <button className="ghost" title={OUTSIDE_COPY.importHint} onClick={() => { setOpen(false); importFile(); }}>
          <FileUp size={15} />{OUTSIDE_COPY.importCid}</button>
      </div>)}
    </div>}
  </span>;
}
