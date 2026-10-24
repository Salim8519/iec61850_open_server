import { useEffect, useRef, useState } from 'react';
import { useReactFlow } from '@xyflow/react';
import { ChevronDown, Plus } from 'lucide-react';
import type { EquipmentNode } from '../../../api/types';
import { GOOSE_IEDS_COPY } from '../../../copy/iedComponent';
import type { Placement } from './iedNodes';
import type { GooseIedsNote } from './useGooseIeds';
import { fitPadding } from '../fitPadding';

const NOTE_MS = 6000;

/** Toolbar part of "Add all GOOSE IEDs" (ADR-0112): the short note after adding (the view moves to
 * the new components), and "N IEDs not on the drawing ▾" with Add per IED while the GOOSE layer is
 * on (the pattern of the outside senders list). */
export function GooseIedsTool({ missing, note, clearNote, addOne, addAll, locked, show }: {
  missing: Placement[]; note: GooseIedsNote | null; clearNote: () => void; addOne: (iedId: string) => void;
  addAll: () => void; locked: boolean; show: boolean;
}) {
  const flow = useReactFlow<EquipmentNode>();
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLSpanElement>(null);
  useEffect(() => {
    if (!note) return undefined;
    const still = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
    // New components are measured after their first paint; then the whole drawing is fitted.
    const fit = note.ids.length ? window.setTimeout(() => flow.fitView({ maxZoom: 1, padding: fitPadding(0.2),
      duration: still ? 0 : 300 }), 250) : undefined;
    const timer = window.setTimeout(clearNote, NOTE_MS);
    return () => { window.clearTimeout(timer); if (fit) window.clearTimeout(fit); };
  }, [note, clearNote, flow]);
  useEffect(() => {
    if (!open) return undefined;
    const close = (event: MouseEvent | KeyboardEvent) => {
      if (event instanceof KeyboardEvent ? event.key === 'Escape' : !root.current?.contains(event.target as Node)) setOpen(false);
    };
    window.addEventListener('mousedown', close); window.addEventListener('keydown', close);
    return () => { window.removeEventListener('mousedown', close); window.removeEventListener('keydown', close); };
  }, [open]);
  const list = show && missing.length > 0;
  if (!list && !note) return null;
  const label = GOOSE_IEDS_COPY.missingButton(missing.length);
  return <span ref={root} className="outside-senders tidy-wrap">
    {list && <button className="ghost" aria-expanded={open} onClick={() => setOpen(!open)} title={GOOSE_IEDS_COPY.missingIntro}>
      {label}<ChevronDown size={15} aria-hidden /></button>}
    {list && open && <div className="outside-popover" role="dialog" aria-label={label}>
      <p className="small muted">{GOOSE_IEDS_COPY.missingIntro}</p>
      {missing.map(ied => <div key={ied.ied_id} className="outside-row"><strong>{ied.name}</strong>
        <button className="ghost" disabled={locked} title={locked ? GOOSE_IEDS_COPY.locked : GOOSE_IEDS_COPY.addHint(ied.name)}
          onClick={() => { setOpen(false); addOne(ied.ied_id); }}><Plus size={15} />{GOOSE_IEDS_COPY.add}</button></div>)}
      {missing.length > 1 && <button className="ghost" disabled={locked} title={GOOSE_IEDS_COPY.addAllHint}
        onClick={() => { setOpen(false); addAll(); }}><Plus size={15} />{GOOSE_IEDS_COPY.addAll}</button>}
    </div>}
    {note && !open && <span className="tidy-note" role="status">{note.text}</span>}
  </span>;
}
