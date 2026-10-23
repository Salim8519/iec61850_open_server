import { GOOSE_IEDS_COPY } from '../../../copy/iedComponent';
import type { PlannedChange } from '../../../equipment/signalPlan';

/** Receivers of a change that have no IED component (ADR-0112): one small chip at the sender,
 * "also TR5 · not on drawing", dashed like everything that is only expected. A click opens the
 * sender's GOOSE links; the toolbar list adds the missing components. */
export function MissingChip({ item, scale, select }: { item: PlannedChange; scale: number; select: () => void }) {
  if (!item.missingAt || !item.missing.length) return null;
  const { x, y } = item.missingAt;
  return <button className="signal-missing nodrag nopan" onClick={select} title={GOOSE_IEDS_COPY.alsoTitle(item.missing)}
    style={{ transform: `translate(${x}px, ${y}px) scale(${scale})` }}>{GOOSE_IEDS_COPY.alsoChip(item.missing)}</button>;
}
