// The type of each logged command (ADR-0095). A gateway log line names the point it operated; the
// point list says what type that point is. A point that is gone from the list stays "unknown".
import type { CommandEntry } from '../../../api/gateway';
import { KIND_LABELS } from '../../../copy/gateway.ts';
import { kindOfGatewayPoint, type Kind } from '../../../points/pointKind.ts';

/** The backend also sends the id of the operated point; the page's CommandEntry type does not list it. */
export type LoggedCommand = CommandEntry;
/** Point id -> gateway point kind (single, command_double …). */
export type PointKinds = ReadonlyMap<string, string>;

export const kindsById = (points: readonly { id?: string; kind: string }[]): PointKinds =>
  new Map(points.flatMap(point => point.id ? [[point.id, point.kind] as const] : []));

export function gatewayCommandKind(entry: LoggedCommand, kinds: PointKinds): Kind {
  const kind = entry.point_id ? kinds.get(entry.point_id) : undefined;
  return kind ? kindOfGatewayPoint(kind) : 'UN';
}

/** The gateway's own name for the operated point's type (for the badge's hover text). */
export function gatewayCommandRaw(entry: LoggedCommand, kinds: PointKinds): string | undefined {
  const kind = entry.point_id ? kinds.get(entry.point_id) : undefined;
  return kind ? (KIND_LABELS as Record<string, string>)[kind] : undefined;
}
