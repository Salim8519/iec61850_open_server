// The point kind of an IEC 61850 signal row (ADR-0095). Pure: the protocol's own types decide
// (FC, basic type, data class), never the name. A Boolean command stays a plain command until its
// data class (SPC or DPC) is known, from the engine's data-class route or the IED's control list.
import type { Control, Signal } from '../api/types';
import { kindOf61850, type Kind } from '../points/pointKind.ts';

export type ClassOf = (ref: string) => string | undefined;
type Typed = Pick<Signal, 'ref' | 'fc' | 'kind'>;

/** SPC / DPC of the control that owns this attribute, from the IED's own control list. */
export function controlClass(controls: readonly Pick<Control, 'ref' | 'cdc'>[] | undefined, ref: string): string | undefined {
  return controls?.find(control => ref === control.ref || ref.startsWith(`${control.ref}.`))?.cdc;
}

/** Kind of one signal; `classOf` answers from the data-class route, `controls` from the IED model. */
export function kindOfSignal(signal: Typed, classOf?: ClassOf, controls?: readonly Pick<Control, 'ref' | 'cdc'>[]): Kind {
  return kindOf61850(signal.fc, signal.kind, classOf?.(signal.ref) ?? controlClass(controls, signal.ref));
}
