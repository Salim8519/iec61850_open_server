// The type of a traffic line, only when the line says it itself (ADR-0095). The 104 gateway and master
// start a line with the standard type identifier (M_DP_TB_1 spont IOA 5 = closed). Other protocols and
// control lines (C_IC_NA_1, STARTDT) carry no point type, so they get no badge: never guessed from names.
import { kindOfIec104, type Kind } from '../../../points/pointKind.ts';

const TYPE_ID = /^([MC]_[A-Z]{2}_[A-Z]{2}_\d)(?=\s|$)/;

/** The point kind and the identifier a traffic line starts with, or null. */
export function trafficKind(text: string): { kind: Kind; token: string } | null {
  const token = TYPE_ID.exec(text)?.[1];
  if (!token) return null;
  const kind = kindOfIec104(token);
  return kind === 'UN' ? null : { kind, token };
}
