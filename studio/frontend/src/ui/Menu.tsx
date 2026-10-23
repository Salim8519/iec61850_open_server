import { useEffect, useLayoutEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { closesOnAway } from './menuAway';
import { menuPlace } from './menuPlace';
import { boxIn, viewIn, zoomOf } from './zoomSpace';
import './ui.css';

export type MenuItem = { label: string; icon?: ReactNode; onSelect: () => void; disabled?: boolean };

/** A button that opens a small list of actions; closes on choice, Escape, outside click or scroll.
 *  The list opens above the page (not inside its panel), so a narrow panel never cuts it off. */
export function Menu({ trigger, items, label, align = 'right', up = false, triggerClass, triggerTour }: {
  trigger: ReactNode; items: MenuItem[]; label: string; align?: 'left' | 'right'; up?: boolean;
  triggerClass?: string; triggerTour?: string;
}) {
  const [open, setOpen] = useState(false);
  const [place, setPlace] = useState<CSSProperties | null>(null);
  const root = useRef<HTMLDivElement>(null);
  const list = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    if (!open || !root.current || !list.current) return setPlace(null);
    // The list is placed in its own zoom space (fit to window), so screen values are converted into it.
    const zoom = zoomOf(list.current);
    const menu = boxIn(list.current.getBoundingClientRect(), zoom);
    setPlace(menuPlace(boxIn(root.current.getBoundingClientRect(), zoom), { width: menu.width, height: menu.height },
      viewIn(zoom), align, up));
  }, [open, align, up]);
  useEffect(() => {
    if (!open) return;
    const inside = (target: EventTarget | null) =>
      root.current?.contains(target as Node) || list.current?.contains(target as Node);
    const close = (event: MouseEvent | KeyboardEvent) => {
      if (event instanceof KeyboardEvent ? event.key === 'Escape' : !inside(event.target)) setOpen(false);
    };
    const away = (event: Event) => {
      if (closesOnAway(event.type, event.target, node => !!inside(node))) setOpen(false);
    };
    window.addEventListener('mousedown', close);
    window.addEventListener('keydown', close);
    window.addEventListener('scroll', away, true);
    window.addEventListener('resize', away);
    return () => {
      window.removeEventListener('mousedown', close);
      window.removeEventListener('keydown', close);
      window.removeEventListener('scroll', away, true);
      window.removeEventListener('resize', away);
    };
  }, [open]);
  return <div ref={root} style={{ position: 'relative' }}>
    <button type="button" className={triggerClass} data-tour={triggerTour} aria-haspopup="menu" aria-expanded={open}
      aria-label={label} onClick={() => setOpen(!open)}>{trigger}</button>
    {open && createPortal(<div ref={list} className="popover menu-pop" role="menu" aria-label={label}
      style={place ?? { left: 0, top: 0, visibility: 'hidden' }}>
      {items.map(item => <button key={item.label} role="menuitem" className="menu-item" disabled={item.disabled}
        onClick={() => { setOpen(false); item.onSelect(); }}>{item.icon}{item.label}</button>)}
    </div>, document.body)}
  </div>;
}
