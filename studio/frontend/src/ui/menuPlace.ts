/** Where a menu list opens: next to its button, never past the window edge (8 px margin). */
export type Box = { left: number; top: number; right: number; bottom: number };
export type Size = { width: number; height: number };

const MARGIN = 8;
const GAP = 6;

export function menuPlace(button: Box, menu: Size, view: Size, align: 'left' | 'right', up: boolean) {
  const width = Math.min(menu.width, view.width - 2 * MARGIN);
  const height = Math.min(menu.height, view.height - 2 * MARGIN);
  const wanted = align === 'left' ? button.left : button.right - width;
  const left = Math.max(MARGIN, Math.min(wanted, view.width - width - MARGIN));
  const below = button.bottom + GAP;
  const above = button.top - GAP - height;
  const fitsBelow = below + height <= view.height - MARGIN;
  const fitsAbove = above >= MARGIN;
  let top = up ? (fitsAbove || !fitsBelow ? above : below) : (fitsBelow || !fitsAbove ? below : above);
  top = Math.max(MARGIN, Math.min(top, view.height - height - MARGIN));
  return { left, top, maxWidth: view.width - 2 * MARGIN, maxHeight: view.height - 2 * MARGIN };
}
