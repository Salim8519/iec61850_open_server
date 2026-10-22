// Words of the SCADA master's own point list (ADR-0110). Plain, beginner-friendly English: the master
// only knows addresses and types, like a real SCADA; it is never linked to an IED.
import type { MasterProtocol } from '../api/master';
import type { PointState } from '../api/masterList';

export const LIST_COPY = {
  tab: 'My point list',
  receivedTab: 'Everything received',
  title: 'My point list',
  hint: 'The points your SCADA expects: address, type and name, like the point list of a real control centre. ScadaBench compares them with what the device sends. It never looks at the IEDs to decide what it received.',
  add: 'Add points', copy: 'Copy from gateway',
  copyHint: 'Fills the list from this project\'s gateway point list: only the address, the type as the master sees it, and the name.',
  copyNone: 'The gateway point list of this protocol is empty. Make the gateway points first (Act as: Station gateway).',
  copyNothingNew: 'Every gateway point is already in your list.',
  more: 'More', moreLabel: 'More point list actions: export or import CSV, Excel template',
  exportCsv: 'Export CSV', importCsv: 'Import CSV', template: 'Download Excel template',
  empty: 'Your point list is empty.',
  emptyHint: 'Add the points your SCADA expects, or copy them from the gateway point list. Everything the device sends is in Everything received.',
  count: (count: number) => count === 1 ? '1 point' : `${count.toLocaleString()} points`,
  added: (count: number) => count === 1 ? '1 point added.' : `${count.toLocaleString()} points added.`,
  removed: (count: number) => count === 1 ? '1 point removed.' : `${count.toLocaleString()} points removed.`,
  undo: 'Undo', remove: 'Remove', removeShown: 'Remove shown',
  send: 'Send', sendHint: 'Sends this point\'s command with its saved type, address, select before operate and value. You confirm first.',
  checkIed: 'Check against IED',
  checkHint: 'Test aid, outside the master: for points copied from the gateway, compares what the master received with the value the gateway\'s own mapping says the IED has.',
  checkColumn: 'IED check (test aid)',
  notInList: 'Not in your list',
  notInListHint: 'Points the device sent that your list does not have.',
  addOne: 'Add', addAll: (count: number) => `Add all ${count.toLocaleString()}`,
  noType: 'No list type for this point',
  columns: { state: 'Result', type: 'Type', address: 'Address', name: 'Name', value: 'Received', scale: 'Scale', unit: 'Unit' },
  namePlaceholder: 'add a name',
  stateFilter: 'Result',
  copiedFrom: (address: number) => `Copied from gateway point ${address}`,
};

/** Each result in words with a symbol, so it never depends on colour (ADR-0110). */
export const STATE_WORDS: Record<PointState, { label: string; symbol: string; hint: string }> = {
  received: { label: 'Received', symbol: '✓', hint: 'The device sent this address with this type.' },
  missing: { label: 'Missing', symbol: '✗', hint: 'Nothing arrived at this address after asking the device for every value.' },
  wrong_type: { label: 'Wrong type', symbol: '⚠', hint: 'The address arrived, but as another type.' },
  waiting: { label: 'Waiting', symbol: '…', hint: 'Not connected yet, or the first ask for all values has not finished.' },
  command: { label: 'Command', symbol: '↗', hint: 'A command point. Devices do not report commands: use Send.' },
};

export const ADD_LIST_COPY = {
  title: 'Add points to my list',
  close: 'Close without adding',
  sources: { one: 'One point', range: 'A range', paste: 'Paste from Excel' },
  sourceHints: {
    one: 'Type the address the device uses and pick the type.',
    range: 'Many points of one type at addresses next to each other, for example 10 single points from IOA 1001.',
    paste: 'Paste rows copied from Excel (Name, Address, Type), or upload the filled Excel template or a CSV file.',
  },
  name: 'Name', address: 'Address', type: 'Type', count: 'How many', from: 'First address', namePrefix: 'Names start with',
  namePrefixPlaceholder: 'for example Feeder alarm', scale: 'Scale', offset: 'Offset', unit: 'Unit',
  select: 'Select before operate', value: 'Value to send', check: 'Check', planRange: 'Show the points',
  add: (count: number) => count === 1 ? 'Add 1 point' : `Add ${count.toLocaleString()} points`,
  fixFirst: 'Fix the rows marked ⚠ first: nothing is added until every row is right.',
  pasteLabel: 'Paste rows here', pastePlaceholder: 'Name\tAddress\tType\nFeeder 1 breaker\t20001\tDouble point',
  pasteHint: 'Copy the rows in Excel and paste them here. A title row (Name, Address, Type) helps; without one ScadaBench guesses the columns.',
  readRows: 'Check the rows', reading: 'Checking the rows…', emptyPaste: 'Paste some rows first.',
  download: 'Download template (.xlsx)', upload: 'Upload filled template', uploadCsv: 'Upload CSV',
  templateHint: 'An Excel file made for this protocol: a Type drop-down on every row. There is no IED column.',
  columnsFound: (names: string) => `Columns found: ${names}.`,
  row: 'Row', typeChoose: 'Choose the type',
};

/** What each protocol calls the address, for the form labels. */
export const LIST_ADDRESS: Record<MasterProtocol, string> = { iec104: 'IOA', dnp3: 'Index', modbus: 'Address' };
