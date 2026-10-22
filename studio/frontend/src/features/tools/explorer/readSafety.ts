import type { Attribute } from '../../../api/tools';

/** Read safety in IED Explorer (change 0177). Reading <DO>.SBO of a "select before operate, normal
 *  security" control IS a Select (IEC 61850-8-1): the device reserves the switch for this PC. So Explorer
 *  never reads SBO (the engine refuses it too), and opening a data object reads no command field (FC CO):
 *  those hold only what the last command sent, not a live value. */
export const READ_SAFETY_COPY = {
  sbo: 'Not read: reading SBO would select the control',
  sboWatch: 'Not added: reading SBO would select the control',
  command: 'Command field, not read',
  commandHint: 'Command fields (FC CO) hold what the last command sent. Explorer does not read them when you open the object, so it never reserves a control by reading.',
};

type Leaf = Pick<Attribute, 'fc' | 'ref'>;

/** True when reading this attribute would select the control: FC CO and the last name SBO. */
export function selectsOnRead(attribute: Leaf): boolean {
  if (attribute.fc.trim().toUpperCase() !== 'CO') return false;
  const name = attribute.ref.split(/[.$]/).pop() ?? '';
  return name.toUpperCase() === 'SBO';
}

/** True for a command field (FC CO), which opening a data object does not read. */
export const isCommandField = (attribute: Leaf) => attribute.fc.trim().toUpperCase() === 'CO';

/** The attributes read when a data object is opened or read again: everything except command fields. */
export function bulkReadable<T extends Leaf>(attributes: T[]): T[] {
  return attributes.filter(attribute => !isCommandField(attribute));
}
