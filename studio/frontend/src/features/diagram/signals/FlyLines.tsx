import { edgePoint, linkCurve } from '../../../equipment/signalLayout';
import type { PlannedChange } from '../../../equipment/signalPlan';
import { EDGE_PULSE_MS } from '../../../equipment/signalPulses';
import { PulseDot } from './PulseDot';
import type { SignalSelection } from './SignalOverlay';

/** A sender with no place on the drawing (outside the project, or no IED component and no trigger):
 * one dot flies from a fixed screen distance beside each receiving IED component into it for ~2 s,
 * with a fading line, then nothing remains (changes 0097, 0102; ADR-0112 targets components). */
export function FlyLines({ plan, now, scale, select }: {
  plan: PlannedChange[]; now: number; scale: number; select: (selection: SignalSelection) => void;
}) {
  return <>{plan.flatMap(change => change.fly).map(line => {
    // Dot and line keep their screen size when zoomed out (change 0102).
    const curve = linkCurve(edgePoint(line.to, scale), line.to);
    return <g key={line.key} className={`signal-link ${line.kind} edge`} onClick={() => select({ kind: 'link', id: line.linkId })}>
      <title>{line.title}</title>
      <path d={curve.path} className="signal-link-hit" />
      <path d={curve.path} className="signal-link-line" />
      {now - line.start < EDGE_PULSE_MS + 100 && <PulseDot curve={curve} kind={line.kind} start={line.start}
        duration={EDGE_PULSE_MS} radius={5 * scale} />}
    </g>;
  })}</>;
}
