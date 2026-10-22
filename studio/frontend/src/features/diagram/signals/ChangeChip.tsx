import { changeChip } from '../../../copy/signalLayers';
import type { PlannedChange } from '../../../equipment/signalPlan';

/** The one plain chip of a change, just above its origin (ADR-0103): "Open · sent to 4 IEDs"
 * (dashed border and →: expected) or "Open · 4 of 4 IEDs got it" (solid border and ✓: seen).
 * Never cut; full words on hover; a click opens the publisher's GOOSE links. */
export function ChangeChip({ item, equipment, scale, select }: {
  item: PlannedChange; equipment: Record<string, { value: unknown; at: number }>; scale: number; select: () => void;
}) {
  const words = changeChip(item.group, equipment);
  const { x, y } = item.chip;
  return <button className={`signal-chip nodrag nopan ${item.group.kind}`} onClick={select} title={words.title}
    aria-label={`${words.full}: ${words.text}`}
    style={{ transform: `translate(${x}px, ${y}px) scale(${scale}) translateY(-100%)` }}>
    <span aria-hidden>{item.group.kind === 'observed' ? '✓' : '→'}</span>{words.text}
  </button>;
}
