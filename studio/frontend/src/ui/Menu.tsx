import { useEffect, useRef, useState, type ReactNode } from 'react';
import './ui.css';

export type MenuItem = { label: string; icon?: ReactNode; onSelect: () => void; disabled?: boolean };

/** A button that opens a small list of actions; closes on choice, Escape or outside click. */
export function Menu({ trigger, items, label, align = 'right' }: {
  trigger: ReactNode; items: MenuItem[]; label: string; align?: 'left' | 'right';
}) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const close = (event: MouseEvent | KeyboardEvent) => {
      if (event instanceof KeyboardEvent ? event.key === 'Escape' : !root.current?.contains(event.target as Node)) setOpen(false);
    };
    window.addEventListener('mousedown', close);
    window.addEventListener('keydown', close);
    return () => { window.removeEventListener('mousedown', close); window.removeEventListener('keydown', close); };
  }, [open]);
  return <div ref={root} style={{ position: 'relative' }}>
    <button type="button" aria-haspopup="menu" aria-expanded={open} aria-label={label} onClick={() => setOpen(!open)}>{trigger}</button>
    {open && <div className="popover" role="menu" style={{ [align]: 0, top: 'calc(100% + 6px)', minWidth: 220 }}>
      {items.map(item => <button key={item.label} role="menuitem" className="menu-item" disabled={item.disabled}
        onClick={() => { setOpen(false); item.onSelect(); }}>{item.icon}{item.label}</button>)}
    </div>}
  </div>;
}
