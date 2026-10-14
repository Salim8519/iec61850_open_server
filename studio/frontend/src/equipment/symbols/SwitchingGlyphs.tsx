import type { Paint } from '../stateColors';

/* Switching symbols drawn for reading at a distance (ADR-0045): a large breaker body
   whose conductor breaks when open, and long disconnector/earth blades with a wide swing.
   Geometry always shows the state; `paint` only repeats it in colour. */

type Pose = 'Open' | 'Closed' | 'Intermediate';
const SWING: Record<Pose, number> = { Closed: 0, Intermediate: 20, Open: 40 };
const BRIDGE: Record<Pose, number> = { Closed: 1, Intermediate: 0.45, Open: 0 };

/** Breaker: filled body = closed, hollow body = open; the conductor inside is whole or broken. */
export function BreakerGlyph({ pose, animate, paint }: { pose?: Pose; animate: boolean; paint?: Paint }) {
  const ink = paint?.ink || 'currentColor';
  return <>
    <path d="M30 0V14M30 46V60" />
    <rect x="14" y="14" width="32" height="32" rx={paint ? 3 : 0} fill={paint?.fill || 'none'}
      stroke={paint?.outline || 'currentColor'} />
    {pose && <g stroke={ink}>
      <path d="M30 14V20M30 40V46" />
      <path className={animate ? 'switch-contact' : ''} d="M30 19V41"
        style={{ transform: `scaleY(${BRIDGE[pose]})`, transformOrigin: '30px 30px', opacity: pose === 'Open' ? 0 : 1 }} />
    </g>}
  </>;
}

/** Disconnector: IEC contact bar on top, a long blade hinged at the bottom terminal. */
export function DisconnectorGlyph({ pose, animate, paint }: { pose?: Pose; animate: boolean; paint?: Paint }) {
  return <>
    <path d="M30 0V12M22 12H38M30 48V60" />
    {pose && <Blade pivot={48} angle={SWING[pose]} animate={animate} paint={paint} />}
    <circle cx="30" cy="48" r="2.6" fill={paint?.fill || 'white'} stroke={paint?.outline || 'currentColor'} />
  </>;
}

/** Earth switch: contact bar on the line side, blade hinged on the earth symbol. */
export function EarthSwitchGlyph({ pose, animate, paint }: { pose?: Pose; animate: boolean; paint?: Paint }) {
  return <>
    <path d="M30 0V12M22 12H38M30 44V49M17 49H43M22 54H38M27 59H33" />
    {pose && <Blade pivot={44} angle={SWING[pose]} animate={animate} paint={paint} />}
    <circle cx="30" cy="44" r="2.6" fill={paint?.fill || 'white'} stroke={paint?.outline || 'currentColor'} />
  </>;
}

/** Unknown or Invalid position: a large cross that never resembles Open. */
export function UnknownCross({ invalid }: { invalid: boolean }) {
  return <path d="M22 22L38 38M38 22L22 38" strokeDasharray={invalid ? undefined : '3 3'} />;
}

/** The moving blade: a bold line, or an outlined bar when painted with a state colour. */
function Blade({ pivot, angle, animate, paint }: { pivot: number; angle: number; animate: boolean; paint?: Paint }) {
  const className = animate ? 'switch-contact' : undefined;
  const style = { transform: `rotate(${angle}deg)`, transformOrigin: `30px ${pivot}px` };
  if (!paint) return <path className={className} style={style} d={`M30 ${pivot - 1}V14`} />;
  return <rect className={className} style={{ ...style, strokeWidth: 'calc(var(--sym-stroke, 3px) * 0.8)' }}
    x="25.5" y="13" width="9" height={pivot - 13} rx="4.5" fill={paint.fill} stroke={paint.outline} />;
}
