import type { EquipmentKind, InfeedType } from '../../api/types';
import type { SwitchPosition } from '../position';
import type { Paint } from '../stateColors';
import { BreakerGlyph, DisconnectorGlyph, EarthSwitchGlyph, UnknownCross } from './SwitchingGlyphs';
import './switch-motion.css';

/** `paint` colours only the part that shows state (breaker body, blade, relay body, button cap).
    Stroke width comes from `--sym-stroke` so the canvas can thicken symbols when zoomed out. */
export function EquipmentSymbol({ kind, closed = false, size = 54, position, animate = false, operated, paint, infeed = 'line' }: {
  kind: EquipmentKind; closed?: boolean; size?: number; position?: SwitchPosition; animate?: boolean;
  operated?: boolean; paint?: Paint; infeed?: InfeedType;
}) {
  const state = position || (closed ? 'Closed' : 'Open');
  const pose = state === 'Open' || state === 'Closed' || state === 'Intermediate' ? state : undefined;
  const glyph = { pose, animate, paint };
  return <svg width={size} height={size} viewBox="0 0 60 60" fill="none" stroke="currentColor"
    style={{ strokeWidth: 'var(--sym-stroke, 3px)' }} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    {kind === 'CB' && <BreakerGlyph {...glyph} />}
    {kind === 'DS' && <DisconnectorGlyph {...glyph} />}
    {kind === 'ES' && <EarthSwitchGlyph {...glyph} />}
    {['CB', 'DS', 'ES'].includes(kind) && !pose && <UnknownCross invalid={state === 'Invalid'} />}
    {kind === 'CT' && <><path d="M30 0V60M44 30H55" /><circle cx="30" cy="30" r="14" /></>}
    {kind === 'VT' && <><path d="M30 0V12M30 48V60" /><circle cx="30" cy="24" r="12" /><circle cx="30" cy="36" r="12" /></>}
    {kind === 'SOURCE' && <><path d="M30 0V12M30 48V60" /><circle cx="30" cy="30" r="18" /><path d="M18 30C21 17 26 17 30 30S39 43 42 30" /></>}
    {kind === 'RELAY' && <><rect x="11" y="12" width="38" height="36" rx="3" /><path d="M19 23H41M19 30H32M19 37H36" /></>}
    {kind === 'TRIP_RELAY' && <><path d="M30 0V10M30 50V60" />
      <rect x="12" y="10" width="36" height="40" rx="4" fill={paint?.fill || 'none'}
        stroke={paint?.outline || 'currentColor'} />
      <g stroke={paint?.ink || 'currentColor'}>
        <circle cx="30" cy="30" r="9" /><path d={operated ? 'M24 30L29 35L38 24' : 'M24 30H36'} />
        <path d="M18 17H42" strokeDasharray={operated === undefined ? '2 2' : undefined} /></g></>}
    {kind === 'PUSH_BUTTON' && <><rect x="13" y="14" width="34" height="32" rx="5" />
      <circle cx="30" cy="30" r={operated ? 9 : 11} fill={paint?.fill || 'none'}
        stroke={paint?.outline || 'currentColor'} />
      <path d="M24 30H36" stroke={paint?.ink || 'currentColor'}
        strokeDasharray={operated === undefined ? '2 2' : undefined} /></>}
    {kind === 'JUNCTION' && <circle cx="30" cy="30" r="9" fill="currentColor" stroke="none" />}
    {kind === 'INFEED' && <InfeedGlyph type={infeed} />}
  </svg>;
}

/** Where power enters: an incoming line, a power transformer or the external grid. */
function InfeedGlyph({ type }: { type: InfeedType }) {
  if (type === 'transformer') return <><path d="M30 0V8M30 52V60" /><circle cx="30" cy="22" r="13" />
    <circle cx="30" cy="38" r="13" /><path d="M25 42L30 34L35 42Z" /></>;
  if (type === 'grid') return <><path d="M30 0V16" /><rect x="14" y="16" width="32" height="28" />
    <path d="M14 30L28 16M14 44L42 16M28 44L46 26" /></>;
  return <><path d="M30 0V50M20 40L30 52L40 40" /><path d="M20 20H40M23 26H37" /></>;
}
