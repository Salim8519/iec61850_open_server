// The point kind of a data attribute in IED Explorer (ADR-0095). Explorer reads MMS type words
// ("float", "bit string"), not the SCL basic types, so this maps them first. The kind comes from the
// functional constraint and the type, never from a name: only the two quality / double-point bit
// strings are told apart by their standard attribute name (q, stVal), because MMS gives no better hint.
import type { Attribute, WatchItem } from '../../../api/tools';
import { kindOf61850, type Kind } from '../../../points/pointKind.ts';

const leafOf = (path: string) => path.split('.').pop() ?? path;

/** The SCL basic type that goes with an MMS type word; anything else stays as it is (and so Unknown). */
export function basicTypeOf(mms: string, leaf: string): string {
  switch (mms.toLowerCase()) {
    case 'boolean': return 'BOOLEAN';
    case 'float': case 'floating-point': return 'FLOAT32';
    case 'integer': return 'INT32';
    case 'unsigned': return 'INT32U';
    case 'visible string': return 'VisString255';
    case 'unicode string': return 'Unicode255';
    case 'octet string': return 'Octet64';
    case 'utc time': case 'generalized time': case 'binary time': return 'Timestamp';
    case 'bit string': return leaf === 'q' ? 'Quality' : leaf === 'stVal' ? 'Dbpos' : 'BitString';
    default: return mms;
  }
}

/** The kind of one attribute. In a CO structure only the control value (ctlVal) is the command; origin,
 *  control number, time, test and check are helpers of that command. `dataClass` (SPC or DPC) is given
 *  when the status value next to the control tells which one it is. */
export function attributeKind(fc: string, path: string, mms: string, dataClass?: string): Kind {
  const leaf = leafOf(path);
  if (fc === 'CO' && leaf !== 'ctlVal') return 'AT';
  return kindOf61850(fc, basicTypeOf(mms, leaf), dataClass);
}

/** SPC and DPC both send a BOOLEAN control value; the status value of the same object tells them apart. */
function controlClass(attributes: readonly Attribute[]): string | undefined {
  const status = attributes.find(attribute => attribute.fc === 'ST' && leafOf(attribute.path) === 'stVal');
  const mms = status?.type.toLowerCase();
  return mms === 'bit string' ? 'DPC' : mms === 'boolean' ? 'SPC' : undefined;
}

/** The kind of every attribute of one data object, in the same order as its attributes. */
export function objectKinds(attributes: readonly Attribute[]): Kind[] {
  const dataClass = controlClass(attributes);
  return attributes.map(attribute => attributeKind(attribute.fc, attribute.path, attribute.type, dataClass));
}

/** A watched value knows only its reference, FC and type, so a boolean control stays a plain command. */
export const watchKind = (item: Pick<WatchItem, 'ref' | 'fc' | 'type'>): Kind => attributeKind(item.fc, item.ref, item.type);
