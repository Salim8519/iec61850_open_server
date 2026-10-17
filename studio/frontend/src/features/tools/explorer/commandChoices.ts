import type { Attribute } from '../../../api/tools';

export type Choice = { label: string; value: boolean | number };

/** The values a user can send to one controllable data object, from its Oper.ctlVal type. */
export function commandChoices(attributes: Attribute[]): Choice[] | 'number' | null {
  const control = attributes.find(attribute => attribute.fc === 'CO' && attribute.path === 'Oper.ctlVal');
  if (!control) return null;
  // Explorer shows MMS type names: a double-point position (DPC) has a bit-string stVal.
  if (control.type.toLowerCase() !== 'boolean') return 'number';
  const position = attributes.some(attribute => attribute.path === 'stVal' && attribute.type.toLowerCase() === 'bit string');
  return position ? [{ label: 'Close', value: true }, { label: 'Open', value: false }]
    : [{ label: 'On', value: true }, { label: 'Off', value: false }];
}
