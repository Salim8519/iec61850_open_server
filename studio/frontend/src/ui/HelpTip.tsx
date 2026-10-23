import { CircleHelp } from 'lucide-react';
import { useId, useRef, useState, type CSSProperties } from 'react';
import { boxIn, viewIn, zoomOf } from './zoomSpace';
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
    if (!button.current) return;
    // The bubble shares the button's zoom (fit to window): screen values are converted into it.
    const zoom = zoomOf(button.current);
    const box = boxIn(button.current.getBoundingClientRect(), zoom);
    const view = viewIn(zoom);
    const left = Math.max(8, Math.min(box.left - 12, view.width - WIDTH - 8));
    const below = box.bottom + 8 + 160 < view.height;
    setPlace(below ? { left, top: box.bottom + 8, width: WIDTH } : { left, top: 'auto', bottom: view.height - box.top + 8, width: WIDTH });
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
