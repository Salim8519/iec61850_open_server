import { changeChip } from '../../../copy/signalLayers';
import { staggerOf, type PlannedChange } from '../../../equipment/signalPlan';
import { RouteLine } from './RouteLine';
import type { SignalSelection } from './SignalOverlay';

type Equipment = Record<string, { value: unknown; at: number }>;

/** The SVG part of each change (ADR-0103): a brief outline on the origin symbol and one smart
 * line per receiving IED. Clicking a line opens that link's details in the side sheet. */
export function ChangeLines({ plan, animated, equipment, scale, select }: {
  plan: PlannedChange[]; animated: Set<number>; equipment: Equipment; scale: number;
  select: (selection: SignalSelection) => void;
}) {
  return <>{plan.map(item => {
    const words = changeChip(item.group, equipment);
    const box = item.box;
    const pad = 4 * scale;
    return <g key={item.group.id} className={`signal-change ${item.group.kind}`}>
      {box && <rect className="signal-origin" x={box.x - pad} y={box.y - pad} width={box.width + 2 * pad}
        height={box.height + 2 * pad} rx={6} />}
      {item.lines.map((line, index) => <RouteLine key={line.key} id={line.key} route={line.route} kind={line.kind}
        start={line.start} delay={staggerOf(index)} animate={animated.has(line.key)} scale={scale}
        title={`${words.full} → ${line.subscriber}${line.kind === 'observed' ? ' · new state seen' : ' · expected'}\nClick for this link`}
        select={() => select({ kind: 'link', id: line.linkId })} />)}
    </g>;
  })}</>;
}
