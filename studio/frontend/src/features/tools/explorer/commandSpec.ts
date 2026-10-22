import type { Attribute, ReadValue } from '../../../api/tools';
import { commandChoices, type Choice } from './commandChoices.ts';

/** How a value is sent (ADR-0104): on/off, a whole number, a listed value, a number or a step. */
export type CommandKind = 'bool' | 'int' | 'enum' | 'float' | 'step';
export type Labelled = { label: string; value: number };
export type CommandSpec = {
  kind: CommandKind; choices: Choice[]; labels: Labelled[];
  model: number | null; sboMs: number | null; min: number | null; max: number | null;
};

/** The standard behaviour modes (IEC 61850-7-4 Mod/Beh). Over MMS an enumeration is only a number, so
 *  Explorer can name the values only for these standard objects; other lists get a number box. */
export const MODE_LABELS: Labelled[] = [{ label: 'On', value: 1 }, { label: 'On, blocked', value: 2 },
  { label: 'Test', value: 3 }, { label: 'Test, blocked', value: 4 }, { label: 'Off', value: 5 }];
const MODE_OBJECTS = new Set(['Mod']);

const numberOf = (value: ReadValue | undefined): number | null =>
  value && typeof value.raw === 'number' && Number.isFinite(value.raw) ? value.raw : null;

/** What one controllable data object takes, from the MMS types Explorer shows and the CF values read. */
export function commandSpec(objectName: string, attributes: Attribute[], values: Record<string, ReadValue>): CommandSpec | null {
  const control = attributes.find(attribute => attribute.fc === 'CO' &&
    (attribute.path === 'Oper.ctlVal' || attribute.path === 'Oper.ctlVal.f'));
  if (!control) return null;
  const read = (...paths: string[]) => {
    for (const path of paths) {
      const attribute = attributes.find(item => item.path === path);
      const number = attribute ? numberOf(values[attribute.ref]) : null;
      if (number !== null) return number;
    }
    return null;
  };
  const type = control.type.toLowerCase();
  let kind: CommandKind = 'int';
  if (control.path === 'Oper.ctlVal.f') kind = 'float';
  else if (type === 'boolean') kind = 'bool';
  else if (type === 'bit string') kind = 'step';
  else if (type === 'float' || type === 'floating-point') kind = 'float';
  else if (MODE_OBJECTS.has(objectName)) kind = 'enum';
  const choices = kind === 'bool' ? (commandChoices(attributes) as Choice[]) : [];
  return { kind, choices, labels: kind === 'enum' ? MODE_LABELS : [],
    model: read('ctlModel'), sboMs: read('sboTimeout'), min: read('minVal', 'minVal.f'), max: read('maxVal', 'maxVal.f') };
}

/** Select before operate: control models 2 (normal) and 4 (enhanced security). */
export const needsSelect = (model: number | null) => model === 2 || model === 4;

/** The control model in plain words. */
export function modelText(model: number | null): string {
  switch (model) {
    case 0: return 'Status only (cannot be operated)';
    case 1: return 'Direct';
    case 2: return 'Select before operate';
    case 3: return 'Direct, waits for the end of the command';
    case 4: return 'Select before operate, waits for the end of the command';
    default: return 'Control model not read yet';
  }
}

/** A typed number is checked here only for being a number; the device checks its own limits. */
export function parsedNumber(kind: CommandKind, text: string): number | null {
  if (text.trim() === '') return null;
  const number = Number(text);
  if (!Number.isFinite(number)) return null;
  return kind === 'int' && !Number.isInteger(number) ? null : number;
}
