// Typed text of an editable cell, turned into the number the point list stores (display helpers only;
// the backend checks every address and scale again when the list is saved).
import type { PointKind, Protocol } from '../../../api/gateway';
import { KIND_LABELS, KIND_WHERE } from '../../../copy/gateway.ts';

/** An address keeps only its digits, as before: "1 042" and "1042" are the same address. */
export const parseAddress = (text: string) => Number(text.replace(/\D/g, ''));

/** A scale is any number (a comma counts as the decimal point); text that is no number becomes 0, which Save refuses. */
export const parseScale = (text: string) => Number(text.trim().replace(',', '.')) || 0;

/** The choices of the type list: the gateway's own name, then how the protocol carries it. */
export const typeChoices = (protocol: Protocol) => (Object.keys(KIND_LABELS) as PointKind[])
  .map(kind => ({ value: kind as string, label: `${KIND_LABELS[kind]} · ${KIND_WHERE[protocol][kind]}` }));
