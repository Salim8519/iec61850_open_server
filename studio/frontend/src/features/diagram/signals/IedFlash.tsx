import type { Box } from '../../../equipment/signalLayout';

/** Short-lived words over an IED component (ADR-0112, from the old IED tags): "✓ got it" after a
 * real stNum increase from a sender on the drawing, "← FEED1 · Open" from a sender with no place on
 * it, and "10.0.0.5 operated XCBR1.Pos" for an outside client's command. They fade in ~2 s; the
 * card itself is never re-rendered by traffic. */
export function IedFlash({ box, scale, received, operated, select }: {
  box: Box; scale: number; received?: { key: number; label: string }; operated?: { key: number; label: string };
  select: () => void;
}) {
  if (!received && !operated) return null;
  return <button className="ied-flash nodrag nopan" onClick={select} title="Click for the GOOSE links of this IED"
    style={{ transform: `translate(${box.x + box.width}px, ${box.y}px) scale(${scale}) translate(-100%, calc(-100% - 4px))` }}>
    {received && <span key={received.key} className="ied-flash-line">{received.label}</span>}
    {operated && <span key={operated.key} className="ied-flash-line">{operated.label}</span>}
  </button>;
}
