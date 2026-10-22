import { useEffect, useRef } from 'react';
import { PULSE_MS, reducedMotion, type PulseKind } from '../../../equipment/signalPulses';
import type { Route } from '../../../equipment/signalRoute';

/** One smart line (ADR-0103). It draws itself behind a moving dot: each frame sets the dot's
 * transform and the reveal mask's stroke-dashoffset (paint only, no layout). Solid line and dot =
 * a real stNum increase; dashed line and hollow dot = expected only. Reduced motion, or more lines
 * than the cap: the whole line shows at once with the dot resting at the receiver, then fades. */
export function RouteLine({ id, route, kind, start, delay, animate, scale, title, select }: {
  id: number; route: Route; kind: PulseKind; start: number; delay: number; animate: boolean; scale: number;
  title: string; select: () => void;
}) {
  const dot = useRef<SVGCircleElement>(null);
  const reveal = useRef<SVGPathElement>(null);
  const still = !animate || reducedMotion();
  useEffect(() => {
    if (still) return;
    let frame = 0;
    const step = () => {
      const t = Math.max(0, Math.min(1, (performance.now() - start) / PULSE_MS));
      const point = route.at(t);
      if (dot.current) {
        dot.current.style.transform = `translate(${point.x}px, ${point.y}px)`;
        dot.current.style.opacity = t <= 0 ? '0' : '1';
      }
      if (reveal.current) reveal.current.style.strokeDashoffset = String(route.length * (1 - t));
      if (t < 1) frame = requestAnimationFrame(step);
    };
    frame = requestAnimationFrame(step);
    return () => cancelAnimationFrame(frame);
  }, [route, start, still]);
  const xs = route.points.map(point => point.x), ys = route.points.map(point => point.y);
  const pad = 12 * scale;
  const area = { x: Math.min(...xs) - pad, y: Math.min(...ys) - pad,
    width: Math.max(...xs) - Math.min(...xs) + 2 * pad, height: Math.max(...ys) - Math.min(...ys) + 2 * pad };
  const rest = route.at(still ? 1 : 0);
  const mask = `signal-reveal-${id}`;
  return <g className={`signal-route ${kind}${still ? ' still' : ''}`} onClick={select}
    style={{ animationDelay: `${delay}ms` }}>
    <title>{title}</title>
    <mask id={mask} maskUnits="userSpaceOnUse" {...area}>
      <path ref={reveal} d={route.path} className="signal-route-reveal"
        style={{ strokeDasharray: `${route.length} ${route.length}`, strokeDashoffset: still ? 0 : route.length }} />
    </mask>
    <path d={route.path} className="signal-link-hit" />
    <path d={route.path} className="signal-route-line" mask={`url(#${mask})`} />
    <circle ref={dot} r={5 * scale} cx={0} cy={0} className={`pulse-dot ${kind}${still ? ' still' : ''}`}
      style={{ transform: `translate(${rest.x}px, ${rest.y}px)`, opacity: still ? 1 : 0 }} />
  </g>;
}
