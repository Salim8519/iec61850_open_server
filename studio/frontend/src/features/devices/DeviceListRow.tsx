import type { ReactNode } from 'react';
import type { Ied } from '../../api/types';
import { roleStatus } from '../../copy/status';
import { deviceBadge, modeStatus } from '../../copy/testMode';
import { StatusPill } from '../../ui/StatusPill';
import type { ModeSummary } from './useModes';

/** One compact IED row in the Devices list: name, data points and its Real/Sim/Off status.
 *  `hints` is the slot for extra per-IED hints (for example GOOSE activity) under the name. */
export function DeviceListRow({ ied, active, mode, select, hints }: {
  ied: Ied; active: boolean; mode?: ModeSummary; select: () => void; hints?: ReactNode;
}) {
  const badge = deviceBadge(mode);
  return <button role="listitem" className={`list-row compact${active ? ' active' : ''}`} aria-current={active || undefined}
    onClick={select}>
    <span className="list-text"><strong>{ied.name}</strong><small>{ied.signal_count.toLocaleString()} data points</small>
      {hints && <span className="list-hints">{hints}</span>}</span>
    <span className="list-pills"><StatusPill status={roleStatus(ied.role, ied.state)} />
      {badge && <StatusPill status={modeStatus(badge)} />}</span>
  </button>;
}
