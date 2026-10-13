export type Side = 'north' | 'east' | 'south' | 'west';
export const SIDES: Side[] = ['north', 'east', 'south', 'west'];
type Connection = { source: string; target: string; sourceHandle?: string | null; targetHandle?: string | null };

/** Terminal sides of `nodeId` that carry at least one wire. */
export function connectedSides(nodeId: string, connections: Connection[]): Set<Side> {
  const sides = new Set<Side>();
  for (const connection of connections) {
    const ends = [connection.source === nodeId ? connection.sourceHandle : null,
      connection.target === nodeId ? connection.targetHandle : null];
    for (const end of ends) if (end && (SIDES as string[]).includes(end)) sides.add(end as Side);
  }
  return sides;
}
