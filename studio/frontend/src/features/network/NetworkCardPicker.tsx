import { ChevronDown, Laptop, Network } from 'lucide-react';
import { useEffect, useLayoutEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { closesOnAway } from '../../ui/menuAway';
import { menuPlace } from '../../ui/menuPlace';
import { boxIn, viewIn, zoomOf } from '../../ui/zoomSpace';
import './picker.css';

export type PickerChoice = { value: string; label: string; tone?: 'loopback' | 'lab'; tag?: string; group: 'pc' | 'cards';
  faint?: boolean };

const ICONS: Record<string, ReactNode> = { loopback: <Laptop size={16} aria-hidden />, lab: <Network size={16} aria-hidden /> };
const GROUPS = { pc: 'On this PC, nothing goes out', cards: 'Real network cards' };

/** The network card list (ADR-0121): a button that opens a list like a dropdown. "This PC only" and the
 *  ScadaBench Lab network sit on top, each with its own colour, icon and tag; the real cards follow.
 *  Arrow keys move, Enter picks, Escape closes; the list opens above the page like Menu. */
export function NetworkCardPicker({ value, choices, placeholder, disabled, onChange }: {
  value: string; choices: PickerChoice[]; placeholder: string; disabled?: boolean; onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const [place, setPlace] = useState<CSSProperties | null>(null);
  const root = useRef<HTMLButtonElement>(null);
  const list = useRef<HTMLDivElement>(null);
  const chosen = choices.find(choice => choice.value === value);
  useLayoutEffect(() => {
    if (!open || !root.current || !list.current) return setPlace(null);
    const zoom = zoomOf(list.current);
    const box = boxIn(root.current.getBoundingClientRect(), zoom);
    const menu = boxIn(list.current.getBoundingClientRect(), zoom);
    setPlace({ ...menuPlace(box, { width: box.width, height: menu.height }, viewIn(zoom), 'left', false), width: box.width });
  }, [open]);
  useEffect(() => {
    if (!open) return;
    const inside = (target: EventTarget | null) =>
      root.current?.contains(target as Node) || list.current?.contains(target as Node);
    const down = (event: MouseEvent) => { if (!inside(event.target)) setOpen(false); };
    const away = (event: Event) => { if (closesOnAway(event.type, event.target, node => !!inside(node))) setOpen(false); };
    window.addEventListener('mousedown', down);
    window.addEventListener('scroll', away, true);
    window.addEventListener('resize', away);
    return () => {
      window.removeEventListener('mousedown', down);
      window.removeEventListener('scroll', away, true);
      window.removeEventListener('resize', away);
    };
  }, [open]);
  const show = () => { setActive(Math.max(0, choices.findIndex(choice => choice.value === value))); setOpen(true); };
  const pick = (choice: PickerChoice) => { setOpen(false); root.current?.focus(); onChange(choice.value); };
  const key = (event: React.KeyboardEvent) => {
    if (!open) {
      if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(event.key)) { event.preventDefault(); show(); }
      return;
    }
    if (event.key === 'Escape') { event.preventDefault(); setOpen(false); }
    else if (event.key === 'ArrowDown') { event.preventDefault(); setActive(now => Math.min(choices.length - 1, now + 1)); }
    else if (event.key === 'ArrowUp') { event.preventDefault(); setActive(now => Math.max(0, now - 1)); }
    else if ((event.key === 'Enter' || event.key === ' ') && choices[active]) { event.preventDefault(); pick(choices[active]); }
  };
  return <>
    <button ref={root} type="button" className={`picker-field${chosen?.tone ? ` tone-${chosen.tone}` : ''}`}
      disabled={disabled} aria-haspopup="listbox" aria-expanded={open} aria-label="Network card"
      onClick={() => (open ? setOpen(false) : show())} onKeyDown={key}>
      {chosen?.tone && ICONS[chosen.tone]}
      <span className="picker-text">{chosen?.label ?? placeholder}</span><ChevronDown size={16} aria-hidden />
    </button>
    {open && createPortal(<div ref={list} className="popover picker-pop" role="listbox" aria-label="Network card"
      style={place ?? { left: 0, top: 0, visibility: 'hidden' }}>
      {choices.map((choice, index) => <div key={choice.value}>
        {(index === 0 || choices[index - 1].group !== choice.group) &&
          <div className="picker-group" role="presentation">{GROUPS[choice.group]}</div>}
        <div role="option" aria-selected={choice.value === value}
          className={`picker-option${choice.tone ? ` tone-${choice.tone}` : ''}${index === active ? ' is-active' : ''}${choice.faint ? ' is-faint' : ''}`}
          onMouseEnter={() => setActive(index)} onMouseDown={event => event.preventDefault()} onClick={() => pick(choice)}>
          {choice.tone && ICONS[choice.tone]}<span className="picker-text">{choice.label}</span>
          {choice.tag && <span className="picker-tag">{choice.tag}</span>}
        </div>
      </div>)}
    </div>, document.body)}
  </>;
}
