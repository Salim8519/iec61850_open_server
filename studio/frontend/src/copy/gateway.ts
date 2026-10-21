// Wording for the gateway simulator (Tools). Plain words first, protocol terms explained.
import type { PointKind, Protocol } from '../api/gateway';

export const PROTOCOL_NAMES: Record<Protocol, string> = { iec104: 'IEC 60870-5-104', modbus: 'Modbus TCP', dnp3: 'DNP3' };

export const PROTOCOL_HINTS: Record<Protocol, string> = {
  iec104: 'The usual protocol between a substation gateway and the control centre. ScadaBench is the controlled station (104 slave); your SCADA connects as the controlling station (104 master) to the port below.',
  modbus: 'Simple register protocol used by many PLCs and smaller SCADA systems. ScadaBench is the Modbus server; your SCADA is the Modbus client. Modbus has no quality flags; an invalid measurement is sent as NaN.',
  dnp3: 'Common in North American utilities. ScadaBench is the outstation; your SCADA is the master and polls or receives events.',
};

export const KIND_LABELS: Record<PointKind, string> = {
  single: 'Single point', double: 'Double point', float: 'Measurement', integer: 'Integer measurement',
  command_single: 'Single command', command_double: 'Double command',
};

export const ADDRESS_LABEL: Record<Protocol, string> = { iec104: 'IOA', modbus: 'Register / coil', dnp3: 'Index' };

export const KIND_WHERE: Record<Protocol, Record<PointKind, string>> = {
  iec104: { single: 'M_SP', double: 'M_DP', float: 'M_ME_NC (float)', integer: 'M_ME_NB (scaled)',
    command_single: 'C_SC', command_double: 'C_DC' },
  modbus: { single: 'Discrete input', double: 'Input/holding register', float: '2 registers, float', integer: '2 registers, int32',
    command_single: 'Coil', command_double: 'Coil (ON = close)' },
  dnp3: { single: 'Binary input', double: 'Double-bit input', float: 'Analog input', integer: 'Analog input',
    command_single: 'Binary output (CROB)', command_double: 'Binary output (CROB, close/trip)' },
};

export const SETTING_LABELS: Record<string, string> = {
  port: 'TCP port', common_address: 'Common address (ASDU)', unit_id: 'Unit ID', outstation_address: 'Outstation address',
  master_address: 'Master address', t1: 't1 (s)', t2: 't2 (s)', t3: 't3 (s)', k: 'k', w: 'w', unsolicited: 'Unsolicited events',
};
export const MAIN_SETTINGS: Record<Protocol, string[]> = {
  iec104: ['port', 'common_address'], modbus: ['port', 'unit_id'], dnp3: ['port', 'outstation_address', 'master_address'],
};
export const ADVANCED_SETTINGS: Record<Protocol, string[]> = {
  iec104: ['t1', 't2', 't3', 'k', 'w'], modbus: [], dnp3: ['unsolicited'],
};

/** Moving a point to another IED or path (the "Change" row in the point list). */
export const CHANGE_COPY = {
  button: 'Change the IED or path of this point', short: 'Change', cancel: 'Close without changing',
  title: (address: number) => `Change the signal of point ${address}`,
  hint: 'Only signals that fit this type are listed. Press Save point list afterwards.',
  noIeds: 'This project has no IEDs yet. Add an IED first.', chooseIed: 'Choose an IED', control: 'Controllable object',
  chooseControl: 'Choose what this command operates', used: 'used by another point', noControls: 'This IED has no controllable objects',
};

export const GATEWAY_COPY = {
  name: 'Name', namePlaceholder: 'Add a name',
  intro: 'ScadaBench acts as the substation gateway. Your SCADA connects to it and sees the IEDs of this project as one station. Press Start gateway: ScadaBench proposes the useful points and waits for SCADA. The IEDs start separately with Run at the top. SCADA commands operate the IEDs over real IEC 61850 MMS.',
  listen: 'Who can connect', thisPc: 'This PC only', network: 'All network adapters',
  listenHint: 'Choose All network adapters when SCADA runs on another PC. ScadaBench checks Windows Firewall for you below.',
  start: 'Start gateway', stop: 'Stop gateway', starting: 'Starting…',
  startHint: 'Press Start gateway. The point list is prepared for you.',
  notRunning: 'The gateway is running, but the IEDs are not: every point is sent to SCADA as invalid until you press Run.',
  separateRun: 'The IEDs are not running. Run and the gateway are separate: start the IEDs with Run, and the gateway with Start gateway.',
  runIeds: 'Run the IEDs', settingsTitle: 'Gateway settings',
  iedsOff: 'The IEDs are not running.',
  skipped: (count: number) => count === 1 ? '1 point skipped' : `${count} points skipped`,
  notBuilt: 'The gateway program is not built in this copy of ScadaBench.',
  saveFirst: 'Save the point list first', commandsOff: 'Commands to real IEDs are off (see the settings)',
  confirmTitle: (label: string, target: string) => `Send "${label}" to ${target}?`,
  realWarning: 'This is a REAL device. The command operates it for real, with its own control model (select before operate when it asks for it). Make sure it is safe to switch and that people on site know.',
  simulatedWarning: 'This operates the simulated IED over IEC 61850 MMS, exactly like a SCADA command. The result appears below and in the Commands tab.',
  cancel: 'Cancel', send: (label: string) => `Yes, send ${label}`,
  sentResult: (label: string, ref: string, ok: boolean, reason: string) =>
    ok ? `"${label}" sent to ${ref}: the IED operated it. The Commands tab shows what changed.` : `"${label}" was not carried out: ${reason}`,
  fromPage: 'You · this page', noCommands: 'No command yet. Commands come from your SCADA or from the Send buttons in Points.',
  realCommands: 'Allow commands to real IEDs',
  realCommandsOn: 'Commands from SCADA and from the Send buttons reach connected real devices too, with their own control model. Switch this off when the real devices must stay untouched.',
  realCommandsOff: 'Real devices are only read. SCADA gets a negative answer for their commands, and their Send buttons are off; simulated IEDs still operate.',
  autoStart: 'The gateway was running when ScadaBench closed, so it starts listening again by itself. The IEDs still start only with Run.',
  connected: (count: number) => count === 1 ? '1 SCADA connected' : `${count} SCADA connections`,
  none: 'No SCADA connected yet',
  tabs: { points: 'Points', traffic: 'Traffic', commands: 'Commands' },
  propose: 'Propose from IEDs', importCsv: 'Import CSV', exportCsv: 'Export CSV', save: 'Save point list', discard: 'Discard changes',
  pointsHint: 'Each row is one SCADA point. ScadaBench proposes only what SCADA usually needs: positions, alarms and trips, measurements, device health and commands. Change any address so it matches your SCADA point list. Add point makes up your own point with a value you control.',
  emptyPoints: 'No points yet. Start the gateway and ScadaBench proposes them, or press Propose from IEDs, or Add point to make up your own, or import your SCADA point list as CSV (type,address,ied,reference,scale,description).',
  lockedPoints: 'Stop the gateway to change the point list.',
  unsaved: 'You have unsaved changes.',
  trafficHint: 'Decoded messages between SCADA and the gateway, newest at the bottom.',
  commandsHint: 'Every command, from SCADA or from this page, in three steps: what was asked, what the IED did, and what ScadaBench saw afterwards.',
  pause: 'Pause', resume: 'Resume', clear: 'Clear view',
  removeAll: 'Remove all points', select: 'Select', selectHint: 'Choose several points to remove',
  removedNote: (count: number) => `${count} ${count === 1 ? 'point' : 'points'} removed. Press Save point list to keep this, or Discard changes to undo.`,
  emptyFilter: 'No row matches the filter. Press Clear to see every row.',
};

// Own points (lane G): values the user makes up, served next to the IED points.
export const OWN_KIND_LABELS: Record<PointKind, string> = {
  single: 'Status (ON / OFF)', double: 'Position (open / closed)', float: 'Measured value', integer: 'Whole number',
  command_single: 'Command (ON / OFF)', command_double: 'Command (open / close)',
};
export const OWN_COPY = {
  add: 'Add point', title: 'Add your own point', name: 'Name', namePlaceholder: 'e.g. Pump running', type: 'Type',
  value: 'Value', scale: 'Scale', drives: 'Changes this status', drivesNone: 'Nothing (only logged)',
  confirm: 'Add to the list', cancel: 'Cancel', ownPoint: 'Own point', newPoint: 'New own point',
  hint: 'An own point is a made-up value you control. SCADA sees it like any other point, with good quality. Change its value here at any time, also while the gateway runs: SCADA gets the change at once.',
  commandHint: 'SCADA can send this command. ScadaBench accepts it, writes it in the Commands tab and sets the status you pick here, so SCADA sees the result. No device is operated.',
  scaleHint: 'SCADA receives the value × scale.',
  addHint: 'Add a made-up point for simulation. The IED points stay as they are.',
  filter: 'Own points', filterHint: 'Only the points you made up yourself.',
  madeUp: 'you set the value', driving: (name: string) => `drives ${name}`,
  saveFirst: 'Save the point list to send the value to SCADA.',
  badNumber: 'Type a number, for example 12.5.', badWhole: 'Type a whole number, for example 42.',
  commandWarning: 'This is your own point: no device is operated. ScadaBench logs the command and sets the status it drives.',
  sentResult: (label: string, name: string, ok: boolean, reason: string) =>
    ok ? `"${label}" sent to ${name}. The Commands tab shows it.` : `"${label}" was not carried out: ${reason}`,
};

// Windows Firewall for the gateway on "All network adapters" (ADR-0062).
export const FIREWALL_COPY = {
  title: 'Windows Firewall',
  checking: 'Checking whether other computers can reach the gateway…',
  open: (port: number) => `Other computers can reach the gateway on TCP port ${port}.`,
  closed: 'Other computers cannot reach the gateway yet: Windows Firewall stops them.',
  blocked: 'Windows made a rule that blocks the gateway, usually when its "allow access?" question was closed.',
  why: 'ScadaBench can fix this for you. It changes only these Windows Firewall rules, and Settings › Network › Clean up undoes them.',
  changes: 'What ScadaBench will change',
  review: 'Allow through Windows Firewall…',
  admin: 'Windows will ask for administrator permission (a UAC prompt). Apply these changes?',
  apply: 'Yes, apply', cancel: 'Cancel',
  failed: 'Windows Firewall could not be checked',
};
