// Words of the gateway's "Add points" dialog and the smart Propose preview (ADR-0105). Plain English first.
import type { Status } from './status';

export const ADD_COPY = {
  open: 'Add points', title: 'Add points', close: 'Close without adding',
  openHint: 'Add points from an IED, made-up points, or rows pasted from Excel.',
  sources: { ied: 'From an IED', simulated: 'Simulated (made up)', paste: 'Paste from Excel' },
  sourceHints: {
    ied: 'Points that show the values of an IED of this project. Commands from SCADA operate that IED.',
    simulated: 'Points whose values you make up and change yourself. No device is involved.',
    paste: 'Copy rows from your SCADA point list in Excel and paste them here, or fill the Excel template.',
  },
  step: (number: number, text: string) => `${number}. ${text}`,
  chooseIed: 'Choose the IED', searchIeds: 'Search IEDs', noIeds: 'This project has no IEDs yet. Import an SCD or CID file first.',
  noIedMatch: 'No IED matches. Clear the search to see all.',
  chooseType: 'Choose the type', mostUsed: 'Most used', moreTypes: 'More types',
  chooseSignals: 'Tick the signals to add', searchSignals: 'Search name or path',
  loadingSignals: 'Loading the signals that fit…', noSignals: 'This IED has nothing that fits this type. Try another type.',
  noSignalMatch: 'Nothing matches the search.', suggested: 'usual',
  suggestedHint: 'Propose from IEDs would pick this one too: SCADA usually needs it.',
  inList: 'in the list', tickShown: 'Tick all shown', untickAll: 'Untick all',
  more: (shown: number, total: number) => `Showing ${shown} of ${total}. Type in the search to find the others.`,
  withStatus: 'Also add its status point',
  withStatusHint: 'SCADA then sees the result of the command. The status gets the matching address, for example 20003 for command 50003.',
  check: 'Planning the addresses…', rowsTitle: 'What will be added',
  rowsHint: 'Names come from the SCL description. Change any name or address here before adding.',
  statusOf: 'status of the command above',
  realLine: 'This is a real device. SCADA commands will operate it.', understand: 'I understand',
  add: (count: number) => count === 1 ? 'Add 1 point' : `Add ${count} points`,
  fixFirst: 'Fix the rows marked in red first. Nothing is added until every row is right.',
  added: (count: number) => count === 1 ? '1 point added.' : `${count} points added.`,
  name: 'Name', address: 'Address', point: 'Point', type: 'Type', ied: 'IED', row: 'Row',
  one: 'One point', range: 'A range', rangeHint: 'Many made-up points of one type, one after the other.',
  count: 'How many', from: 'From address', namePrefix: 'Name (numbered)', namePrefixPlaceholder: 'e.g. Pump',
  planRange: 'Show the points',
  pasteLabel: 'Paste rows from Excel here (Ctrl+V)',
  pastePlaceholder: 'Address\tType\tIED\tPoint\tName\n20001\tDouble point\tFEED1\tFEED1LD0/XCBR1.Pos.stVal\tCB1 position',
  pasteHint: 'A title row is optional. ScadaBench recognises the columns address, type, IED, point (path) and name. Leave IED empty for a made-up point.',
  readRows: 'Read the rows', columnsFound: (names: string) => `Columns found: ${names}.`,
  template: 'Excel template', templateHint: 'The template has drop-down lists: the types of this protocol, your IEDs, and only the points that fit.',
  download: 'Download template (.xlsx)', upload: 'Upload filled template', reading: 'Reading the file…',
  emptyPaste: 'Paste some rows first.',
};

/** The Simulated / Real badge of an IED in the chooser (words + pattern; colour only repeats them). */
export function iedBadge(role: string): Status {
  if (role === 'real') return { label: 'Real device', tone: 'warning', hint: 'Monitored real device: commands operate it for real.' };
  if (role === 'off') return { label: 'Skipped', tone: 'off', hint: 'Skipped in Devices: its points are invalid until it runs.' };
  return { label: 'Simulated', tone: 'info', hint: 'ScadaBench simulates this IED.' };
}

export const SMART_COPY = {
  title: 'Propose from IEDs', hint: 'ScadaBench picks what SCADA usually needs: positions, alarms and trips, measurements, device health and commands. Names come from the SCL; each command gets its status point at the matching address.',
  modes: { add_new: 'Add only new points', replace: 'Replace the IED points' },
  modeHints: {
    add_new: 'Your list stays exactly as it is, with your names and addresses. Only points it does not have yet are added.',
    replace: 'The IED points are proposed again from scratch. Your own made-up points stay.',
  },
  ieds: 'Propose from these IEDs', offered: (offered: number, fresh: number) => `${offered} points · ${fresh} new`,
  loading: 'Looking at the IEDs…', apply: (count: number) => count === 1 ? 'Apply: 1 point' : `Apply: ${count} points`,
  result: (count: number, now: number) => `After this the list has ${now} points (${count} from the ticked IEDs).`,
  noIeds: 'Tick at least one IED.', cancel: 'Cancel',
};
