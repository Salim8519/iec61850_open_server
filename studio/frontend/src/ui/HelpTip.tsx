import { CircleHelp } from 'lucide-react';
import { useId, useRef, useState, type CSSProperties } from 'react';
import './ui.css';

const WIDTH = 300;

/** A "?" next to a label that opens a longer plain-English explanation (mouse, keyboard, touch).
 *  It keeps beginner help one tap away without pushing the list down (change 0093). The bubble
 *  is placed on the window, so scrolling panes and narrow columns never cut it off. */
export function HelpTip({ text, label = 'More about this' }: { text: string; label?: string }) {
  const [place, setPlace] = useState<CSSProperties | null>(null);
  const button = useRef<HTMLButtonElement>(null);
  const id = useId();
  if (!text) return null;
  const show = () => {
    const box = button.current?.getBoundingClientRect();
    if (!box) return;
    const left = Math.max(8, Math.min(box.left - 12, window.innerWidth - WIDTH - 8));
    const below = box.bottom + 8 + 160 < window.innerHeight;
    setPlace(below ? { left, top: box.bottom + 8, width: WIDTH } : { left, top: 'auto', bottom: window.innerHeight - box.top + 8, width: WIDTH });
  };
  const hide = () => setPlace(null);
  return <span className="info-tip help-tip" onMouseLeave={hide}>
    <button ref={button} type="button" className="info-tip-button" aria-label={label} aria-expanded={!!place}
      aria-describedby={place ? id : undefined} onClick={() => place ? hide() : show()} onMouseEnter={show}
      onFocus={show} onBlur={hide} onKeyDown={event => { if (event.key === 'Escape') hide(); }}>
      <CircleHelp size={15} />
    </button>
    {place && <span role="tooltip" id={id} className="info-tip-bubble help-bubble" style={place}>{text}</span>}
  </span>;
}
