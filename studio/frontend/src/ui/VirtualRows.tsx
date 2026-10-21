import { useCallback, useEffect, useLayoutEffect, useRef, useState, type RefObject } from 'react';
import { sameRange, virtualRange, type VirtualRange } from './virtualRange';

type Key = string | number;

/** Virtual rows for a long unpaged list inside the scrolling element `box` (speed lane F).
 *
 * Only the rows on screen (plus a few) are rendered; two spacers keep the scrollbar honest. Each
 * rendered row carries `data-vrow={key}` so its real height is measured (rows may wrap); a row not
 * seen yet counts as `rowHeight`. Put `spacer` on the spacer before the first row. */
export function useVirtualRows(box: RefObject<HTMLElement | null>, count: number, keyOf: (index: number) => Key,
  rowHeight: number, overscan = 8) {
  const heights = useRef(new Map<string, number>());
  const spacer = useRef<HTMLElement | null>(null);
  const frame = useRef(0);
  const [range, setRange] = useState<VirtualRange>(() =>
    virtualRange({ count, rowHeight, scrollTop: 0, viewport: 800, overscan }));
  const measure = useCallback(() => {
    const element = box.current;
    if (!element) return;
    element.querySelectorAll<HTMLElement>('[data-vrow]').forEach(row => {
      heights.current.set(row.dataset.vrow!, row.getBoundingClientRect().height);
    });
    // Where the list starts inside the scrolling element (a header or toolbar may come first).
    const offset = spacer.current
      ? spacer.current.getBoundingClientRect().top - element.getBoundingClientRect().top + element.scrollTop : 0;
    const next = virtualRange({ count, rowHeight, overscan, scrollTop: element.scrollTop - offset,
      viewport: element.clientHeight, heightOf: index => heights.current.get(String(keyOf(index))) });
    setRange(current => sameRange(current, next) ? current : next);
  }, [box, count, keyOf, rowHeight, overscan]);
  // After every render: new rows got measured, so the spacers settle within a frame or two.
  useLayoutEffect(measure);
  useEffect(() => {
    const element = box.current;
    if (!element) return undefined;
    const soon = () => {
      if (frame.current) return;
      frame.current = requestAnimationFrame(() => { frame.current = 0; measure(); });
    };
    element.addEventListener('scroll', soon, { passive: true });
    const resize = new ResizeObserver(soon);
    resize.observe(element);
    return () => {
      element.removeEventListener('scroll', soon);
      resize.disconnect();
      if (frame.current) cancelAnimationFrame(frame.current);
      frame.current = 0;
    };
  }, [box, measure]);
  return { range, spacer };
}

/** Spacer row for a table body; keeps the column layout untouched. */
export function SpacerRow({ height, columns, spacerRef }: {
  height: number; columns: number; spacerRef?: RefObject<HTMLElement | null>;
}) {
  return <tr aria-hidden="true" className="virtual-spacer" ref={spacerRef as RefObject<HTMLTableRowElement>}>
    <td colSpan={columns} style={{ height, padding: 0, border: 0 }} /></tr>;
}

/** Spacer for a list of blocks (div or li). */
export function SpacerBlock({ height, item = false, spacerRef }: {
  height: number; item?: boolean; spacerRef?: RefObject<HTMLElement | null>;
}) {
  return item ? <li aria-hidden="true" className="virtual-spacer" style={{ height }} ref={spacerRef as RefObject<HTMLLIElement>} />
    : <div aria-hidden="true" className="virtual-spacer" style={{ height }} ref={spacerRef as RefObject<HTMLDivElement>} />;
}
