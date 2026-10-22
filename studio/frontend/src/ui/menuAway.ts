/** Whether an open menu closes on a scroll or resize (change 0177). A resize always closes it: its target
 *  is the window, which is not a page element, and asking an element whether it "contains" the window
 *  throws, which left the menu open. A scroll closes it unless it happened inside the menu or its button. */
export function closesOnAway(type: string, target: unknown, inside: (node: Node) => boolean): boolean {
  if (type === 'resize') return true;
  const isNode = typeof Node !== 'undefined' && target instanceof Node;
  return !(isNode && inside(target as Node));
}
