import type { ReactNode } from 'react';
import { Check } from 'lucide-react';
import type { Ied } from '../../api/types';
import { roleStatus } from '../../copy/status';
import { deviceBadge, modeStatus } from '../../copy/testMode';
import { StatusPill } from '../../ui/StatusPill';
import type { ModeSummary } from './useModes';

/** One compact IED row in the Devices list: name, data points and its Real/Sim/Off status.
 *  `hints` is the slot for extra per-IED hints (for example GOOSE activity) under the name. */
export function DeviceListRow({ ied, active, mode, select, hints, checked, live = false }: {
  ied: Ied; active: boolean; mode?: ModeSummary; select: () => void; hints?: ReactNode;
  /** Set while choosing several IEDs: the row shows a tick box and a click toggles it. */
  checked?: boolean;
  /** Running while other IEDs are not (change 0106): a solid bar at the left edge. */
  live?: boolean;
}) {
  const badge = deviceBadge(mode);
  const picking = checked !== undefined;
  return <button role="listitem" className={`list-row compact${active && !picking ? ' active' : ''}${checked ? ' checked' : ''}${live ? ' is-live' : ''}`}
    aria-current={(active && !picking) || undefined} aria-pressed={picking ? checked : undefined} onClick={select}>
    {picking && <span className={`row-check${checked ? ' on' : ''}`} aria-hidden="true">{checked && <Check size={14} />}</span>}
    <span className="list-text"><strong>{ied.name}</strong><small>{ied.signal_count.toLocaleString()} data points</small>
      {/* Status pills get their own line so "Simulated"/"Real device" are never cut (change 0097). */}
      <span className="list-pills"><StatusPill status={roleStatus(ied.role, ied.state)} />
        {badge && <StatusPill status={modeStatus(badge)} />}</span>
      {hints && <span className="list-hints">{hints}</span>}</span>
  </button>;
}
