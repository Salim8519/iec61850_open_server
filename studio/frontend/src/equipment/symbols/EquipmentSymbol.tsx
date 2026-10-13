import type { EquipmentKind } from '../../api/types';
import type { SwitchPosition } from '../position';
import type { Paint } from '../stateColors';
import './switch-motion.css';

/** `paint` colours only the part that shows state (breaker body, blade, relay body, button cap). */
export function EquipmentSymbol({ kind, closed = false, size = 54, position, animate = false, operated, paint }: {
  kind: EquipmentKind; closed?: boolean; size?: number; position?: SwitchPosition; animate?: boolean;
  operated?: boolean; paint?: Paint;
}) {
  const state = position || (closed ? 'Closed' : 'Open');
  const known = ['Open', 'Closed', 'Intermediate'].includes(state);
  const gap = state === 'Closed' ? 0 : state === 'Intermediate' ? 3 : 6;
  const angle = state === 'Closed' ? 0 : state === 'Intermediate' ? 17 : 35;
  return <svg width={size} height={size} viewBox="0 0 60 60" fill="none" stroke="currentColor"
    strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    {kind === 'CB' && <><path d="M30 0V19M30 41V60" />
      <rect x="19" y="19" width="22" height="22" rx={paint ? 3 : 0} fill={paint?.fill || 'none'}
        stroke={paint?.outline || 'currentColor'} strokeWidth={paint ? 2 : undefined} />
      {known && <g stroke={paint?.ink || 'currentColor'} fill={paint?.ink || 'none'}>
        <circle cx="30" cy="24" r={paint ? 1.6 : 1} /><circle cx="30" cy="36" r={paint ? 1.6 : 1} />
        <path className={animate ? 'switch-contact' : ''} style={{ transform: `translateX(${gap}px)` }}
          strokeWidth={paint ? 2.4 : undefined} d="M30 24V36" /></g>}
    </>}
    {kind === 'DS' && <><path d="M30 0V18M30 42V60" /><circle cx="30" cy="19" r="2" />
      {known && <Blade animate={animate} angle={angle} paint={paint} />}
      <circle cx="30" cy="42" r="2" fill={paint?.fill || 'none'} stroke={paint?.outline || 'currentColor'} /></>}
    {kind === 'ES' && <><path d="M30 0V18M30 41V49M18 49H42M22 54H38M27 59H33" />
      {known && <Blade animate={animate} angle={angle} paint={paint} />}
      <circle cx="30" cy="19" r="2" fill={paint ? 'white' : 'none'} stroke="currentColor" /></>}
    {['CB', 'DS', 'ES'].includes(kind) && !known && <path d="M25 25L35 35M35 25L25 35" strokeDasharray={state === 'Invalid' ? undefined : '2 2'} />}
    {kind === 'CT' && <><path d="M30 0V60M44 30H55" /><circle cx="30" cy="30" r="14" /></>}
    {kind === 'VT' && <><path d="M30 0V12M30 48V60" /><circle cx="30" cy="24" r="12" /><circle cx="30" cy="36" r="12" /></>}
    {kind === 'SOURCE' && <><path d="M30 0V12M30 48V60" /><circle cx="30" cy="30" r="18" /><path d="M18 30C21 17 26 17 30 30S39 43 42 30" /></>}
    {kind === 'RELAY' && <><rect x="11" y="12" width="38" height="36" rx="3" /><path d="M19 23H41M19 30H32M19 37H36" /></>}
    {kind === 'TRIP_RELAY' && <><path d="M30 0V10M30 50V60" />
      <rect x="12" y="10" width="36" height="40" rx="4" fill={paint?.fill || 'none'}
        stroke={paint?.outline || 'currentColor'} strokeWidth={paint ? 2 : undefined} />
      <g stroke={paint?.ink || 'currentColor'} strokeWidth={paint ? 2.2 : undefined}>
        <circle cx="30" cy="30" r="9" /><path d={operated ? 'M24 30L29 35L38 24' : 'M24 30H36'} />
        <path d="M18 17H42" strokeDasharray={operated === undefined ? '2 2' : undefined} /></g></>}
    {kind === 'PUSH_BUTTON' && <><rect x="13" y="14" width="34" height="32" rx="5" />
      <circle cx="30" cy="30" r={operated ? 9 : 11} fill={paint?.fill || 'none'}
        stroke={paint?.outline || 'currentColor'} strokeWidth={paint ? 2 : undefined} />
      <path d="M24 30H36" stroke={paint?.ink || 'currentColor'} strokeWidth={paint ? 2.2 : undefined}
        strokeDasharray={operated === undefined ? '2 2' : undefined} /></>}
    {kind === 'JUNCTION' && <circle cx="30" cy="30" r="9" fill="currentColor" stroke="none" />}
  </svg>;
}

/** Switch blade: a thin line normally; a solid outlined bar when painted with a state colour. */
function Blade({ animate, angle, paint }: { animate: boolean; angle: number; paint?: Paint }) {
  const className = animate ? 'switch-contact switch-blade' : 'switch-blade';
  const style = { transform: `rotate(${angle}deg)` };
  if (!paint) return <path className={className} style={style} d="M30 41V21" />;
  return <rect className={className} style={style} x="27" y="20" width="6" height="22" rx="3"
    fill={paint.fill} stroke={paint.outline} strokeWidth="1.4" />;
}
