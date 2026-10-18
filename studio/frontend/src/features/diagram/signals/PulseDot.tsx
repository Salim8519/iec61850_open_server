import { useEffect, useRef } from 'react';
import { PULSE_MS, reducedMotion, type PulseKind } from '../../../equipment/signalPulses';

type Curve = { at: (t: number) => { x: number; y: number } };

/** One dot along a link for ~0.7 s (~2 s from the canvas edge for a sender outside the project), moved with transform only. Solid = seen by the listener,
 * hollow = expected from the publisher's change. Reduced motion: the dot rests at the subscriber. */
export function PulseDot({ curve, kind, start, duration = PULSE_MS }: {
  curve: Curve; kind: PulseKind; start: number; duration?: number;
}) {
  const ref = useRef<SVGCircleElement>(null);
  const still = reducedMotion();
  useEffect(() => {
    if (still) return;
    let frame = 0;
    const step = () => {
      const t = Math.min(1, (performance.now() - start) / duration);
      const point = curve.at(t);
      if (ref.current) {
        ref.current.style.transform = `translate(${point.x}px, ${point.y}px)`;
        ref.current.style.opacity = t >= 1 ? '0' : '1';
      }
      if (t < 1) frame = requestAnimationFrame(step);
    };
    frame = requestAnimationFrame(step);
    return () => cancelAnimationFrame(frame);
  }, [curve, start, still, duration]);
  const rest = curve.at(still ? 1 : 0);
  return <circle ref={ref} r={5} cx={0} cy={0} className={`pulse-dot ${kind}${still ? ' still' : ''}`}
    style={{ transform: `translate(${rest.x}px, ${rest.y}px)` }} />;
}
