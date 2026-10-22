import type { ExtendedControl, PanelPoint } from '../api/types.ts';
import type { CommandControl, CommandValue } from './pointRequests.ts';

/** How a command row is operated (ADR-0108): fixed choices (SPC, DPC), a number box (INC whole
 *  numbers, APC numbers with the row's unit), a list of labels (ENC) or Lower/Raise (BAC, BSC, ISC).
 *  Display only: the Go engine checks range, step and labels again and answers in words. */
export type CommandInput = 'choices' | 'number' | 'list' | 'steps' | 'none';

export function commandInput(control: Pick<CommandControl, 'cdc'> | undefined): CommandInput {
  switch (control?.cdc) {
    case 'SPC': case 'DPC': return 'choices';
    case 'INC': case 'APC': return 'number';
    case 'ENC': return 'list';
    case 'BAC': case 'BSC': case 'ISC': return 'steps';
  }
  return 'none';
}

const typed = (control: CommandControl | undefined): Partial<ExtendedControl> => (control ?? {}) as Partial<ExtendedControl>;

/** "from 1 to 17, steps of 1" for the hint of a number box; '' when the model gives no bounds. */
export function rangeHint(control: CommandControl | undefined): string {
  const { min, max, step } = typed(control);
  const parts = [typeof min === 'number' ? `from ${min}` : '', typeof max === 'number' ? `to ${max}` : '',
    typeof step === 'number' && step > 0 ? `steps of ${step}` : ''].filter(Boolean);
  return parts.join(' ').replace(/ steps/, ', steps');
}

/** The typed number of a number box: a number, and a whole one for INC. */
export function parseCommandNumber(control: CommandControl | undefined, text: string): { value?: number; error?: string } {
  const value = Number(text.trim().replace(',', '.'));
  if (text.trim() === '' || !Number.isFinite(value)) return { error: 'Type a number, for example 5.' };
  if (control?.cdc === 'INC' && !Number.isInteger(value)) return { error: 'This command takes whole numbers only.' };
  return { value };
}

/** The ENC labels in the model's order (the keys of `labels`, label to number). */
export const enumChoices = (control: CommandControl | undefined): string[] => Object.keys(typed(control).labels ?? {});

/** The command in words for messages and the countdown ("5 kV", "blocked", "Raise", "Close"). */
export function commandWords(point: Pick<PanelPoint, 'unit'>, control: CommandControl | undefined, value: CommandValue): string {
  if (value === 'higher') return 'Raise';
  if (value === 'lower') return 'Lower';
  if (typeof value === 'boolean') return control?.cdc === 'DPC' ? (value ? 'Close' : 'Open') : (value ? 'ON' : 'OFF');
  if (typeof value === 'number') return `${value}${control?.cdc === 'APC' && point.unit ? ` ${point.unit}` : ''}`;
  return value;
}

/** Short words for the class in the Add points list ("whole number 0 to 10", "list: on, blocked"). */
export function controlSummary(control: CommandControl): string {
  const range = rangeHint(control);
  switch (control.cdc) {
    case 'DPC': return 'Open / Close';
    case 'SPC': return 'ON / OFF';
    case 'INC': return `whole number${range ? ` ${range}` : ''}`;
    case 'APC': return `set point${range ? ` ${range}` : ''}`;
    case 'ENC': return `list: ${enumChoices(control).join(', ') || 'no labels'}`;
    case 'BAC': case 'BSC': case 'ISC': return 'Lower / Raise';
  }
  return (control as { cdc: string }).cdc;
}
