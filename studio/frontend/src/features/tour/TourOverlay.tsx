import { useCallback, useEffect, useLayoutEffect, useState } from 'react';
import { X } from 'lucide-react';
import type { PlaceId } from '../../app/places';
import { TOUR_COPY as COPY, TOUR_STEPS as STEPS } from '../../copy/tour';
import { boxIn, viewIn, zoomOf } from '../../ui/zoomSpace';
import './tour.css';

type Box = { top: number; left: number; width: number; height: number };
const PAD = 8;
const CARD_WIDTH = 360;

function targetBox(target?: string): Box | null {
  if (!target) return null;
  const element = document.querySelector<HTMLElement>(`[data-tour="${target}"]`);
  if (!element) return null;
  element.scrollIntoView({ block: 'nearest', inline: 'nearest' });
  // The ring sits in the page's zoom (fit to window); the target's screen box is converted into it.
  const rect = boxIn(element.getBoundingClientRect(), zoomOf());
  if (!rect.width && !rect.height) return null;
  return { top: rect.top - PAD, left: rect.left - PAD, width: rect.width + PAD * 2, height: rect.height + PAD * 2 };
}

function cardPosition(box: Box | null): { top: number; left: number } {
  const { width, height } = viewIn(zoomOf());
  if (!box) return { top: Math.max(24, height / 2 - 140), left: Math.max(16, width / 2 - CARD_WIDTH / 2) };
  const right = box.left + box.width + 16;
  if (right + CARD_WIDTH < width) return { top: Math.min(Math.max(16, box.top), height - 260), left: right };
  const below = box.top + box.height + 16;
  return { top: Math.min(below, height - 260), left: Math.min(Math.max(16, box.left), width - CARD_WIDTH - 16) };
}

/** Spotlight tour: dims the app, rings one control, explains it in plain words. */
export function TourOverlay({ go, close }: { go: (place: PlaceId) => void; close: () => void }) {
  const [index, setIndex] = useState(0);
  const [box, setBox] = useState<Box | null>(null);
  const step = STEPS[index];
  const last = index === STEPS.length - 1;
  const measure = useCallback(() => setBox(targetBox(step.target)), [step]);

  useEffect(() => { if (step.place) go(step.place); }, [step, go]);
  useLayoutEffect(() => {
    measure();
    const timer = window.setTimeout(measure, 120);
    window.addEventListener('resize', measure);
    return () => { window.clearTimeout(timer); window.removeEventListener('resize', measure); };
  }, [measure]);
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') close();
      if (event.key === 'ArrowRight' && !last) setIndex(index + 1);
      if (event.key === 'ArrowLeft' && index > 0) setIndex(index - 1);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [index, last, close]);

  const position = cardPosition(box);
  return <div className="tour" role="dialog" aria-modal="true" aria-labelledby="tour-title">
    {box ? <div className="tour-ring" style={box} /> : <div className="tour-dim" />}
    <div className="tour-card" style={{ ...position, width: CARD_WIDTH }}>
      <div className="tour-head">
        <span className="small muted">{COPY.progress(index + 1, STEPS.length)}</span>
        <button className="ghost icon" onClick={close} aria-label={COPY.skip}><X size={16} /></button>
      </div>
      <h2 id="tour-title">{step.title}</h2>
      <p>{step.text}</p>
      <div className="tour-dots" aria-hidden="true">
        {STEPS.map((item, dot) => <span key={item.id} className={dot === index ? 'on' : ''} />)}
      </div>
      <div className="tour-actions">
        {index === 0 ? <button className="ghost" onClick={close}>{COPY.skip}</button>
          : <button onClick={() => setIndex(index - 1)}>{COPY.back}</button>}
        <button className="primary" autoFocus onClick={() => (last ? close() : setIndex(index + 1))}>
          {last ? COPY.done : COPY.next}</button>
      </div>
    </div>
  </div>;
}
