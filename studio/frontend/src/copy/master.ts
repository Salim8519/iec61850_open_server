// Wording for the SCADA roles (ADR-0061). Each protocol keeps its own official words:
// IEC 104 controlling/controlled station, DNP3 master/outstation, Modbus client/server.
import type { CommandKind, CrobCode, MasterProtocol, Role } from '../api/master';

export const ROLE_NAMES: Record<Role, string> = {
  gateway: 'Station gateway', master: 'SCADA master', converter: 'Protocol converter',
};

export const ROLE_HINTS: Record<Role, string> = {
  gateway: 'ScadaBench is the substation gateway. It reads your IEDs and serves them to your SCADA.',
  master: 'ScadaBench is the SCADA. It connects to your gateway, RTU or outstation, reads every point and can send commands.',
  converter: 'ScadaBench reads another gateway or RTU as a master and serves the same points to your SCADA.',
};

/** Why the role buttons are greyed out while something runs (said on every button, not only in the help). */
export const ROLE_LOCKED = 'Stop or disconnect to switch roles.';

/** Tooltip of one role button: what the role does, and why it cannot be picked right now. */
export function roleTitle(role: Role, locked: boolean): string {
  return locked ? `${ROLE_HINTS[role]} ${ROLE_LOCKED}` : ROLE_HINTS[role];
}

/** What ScadaBench is and what the device is, in the words of each protocol. */
export const SIDES: Record<MasterProtocol, { us: string; device: string; usShort: string; deviceShort: string }> = {
  iec104: { us: 'controlling station (104 master)', device: 'controlled station (104 slave)',
    usShort: '104 master', deviceShort: '104 slave' },
  dnp3: { us: 'DNP3 master', device: 'DNP3 outstation', usShort: 'DNP3 master', deviceShort: 'outstation' },
  modbus: { us: 'Modbus client', device: 'Modbus server', usShort: 'Modbus client', deviceShort: 'Modbus server' },
};

export const DIAGRAM = { scada: 'SCADA · control centre', device: 'Gateway · RTU · outstation', ieds: 'IEDs (IEC 61850)',
  here: 'ScadaBench', title: 'Which side am I?' };

export const MASTER_SETTING_LABELS: Record<string, string> = {
  port: 'TCP port', common_address: 'Common address (ASDU)', originator_address: 'Originator address',
  t0: 't0 connect (s)', t1: 't1 (s)', t2: 't2 (s)', t3: 't3 (s)', k: 'k', w: 'w',
  time_tagged_commands: 'Send commands with a time tag (C_SC_TA_1 and similar)',
  gi_minutes: 'Ask for all values every (min)',
  outstation_address: 'Outstation address', master_address: 'Master address',
  integrity_minutes: 'Integrity poll every (min)', event_seconds: 'Event poll every … s',
  unsolicited: 'Accept unsolicited events', unit_id: 'Unit ID', poll_ms: 'Read every … ms',
};
export const MASTER_MAIN: Record<MasterProtocol, string[]> = {
  iec104: ['port', 'common_address', 'gi_minutes'], dnp3: ['port', 'outstation_address', 'master_address'],
  modbus: ['port', 'unit_id', 'poll_ms'],
};
export const MASTER_ADVANCED: Record<MasterProtocol, string[]> = {
  iec104: ['originator_address', 't0', 't1', 't2', 't3', 'k', 'w', 'time_tagged_commands'],
  dnp3: ['integrity_minutes', 'event_seconds', 'unsolicited'], modbus: [],
};

export const COMMAND_NAMES: Record<CommandKind, string> = {
  single: 'Single command (C_SC)', double: 'Double command (C_DC)', setpoint_float: 'Setpoint, float (C_SE_NC)',
  setpoint_scaled: 'Setpoint, scaled (C_SE_NB)', setpoint_normalized: 'Setpoint, normalised (C_SE_NA)',
  clock: 'Clock synchronisation (C_CS)', binary_output: 'Binary output (CROB)',
  analog_output_int: 'Analog output, integer (g41v1)', analog_output_float: 'Analog output, float (g41v3)',
  coil: 'Write coil (FC5)', register: 'Write register (FC6)', registers_float: 'Write float, 2 registers (FC16)',
  registers_int32: 'Write 32-bit integer, 2 registers (FC16)',
};
export const COMMANDS: Record<MasterProtocol, CommandKind[]> = {
  iec104: ['double', 'single', 'setpoint_float', 'setpoint_scaled', 'setpoint_normalized', 'clock'],
  dnp3: ['binary_output', 'analog_output_float', 'analog_output_int'],
  modbus: ['coil', 'register', 'registers_float', 'registers_int32'],
};
export const CROB_CODES: Record<CrobCode, string> = {
  latch: 'Latch on / off', pulse: 'Pulse on', trip_close: 'Trip / close (pulse)',
};
export const ADDRESS_WORD: Record<MasterProtocol, string> = {
  iec104: 'Command address (IOA)', dnp3: 'Output index', modbus: 'Register or coil address',
};
/** What the confirmation calls the target: DNP3 outputs have an index (change 0089). */
export const TARGET_WORD: Record<MasterProtocol, string> = { iec104: 'address', dnp3: 'index', modbus: 'address' };
export const COMMAND_HINT: Record<MasterProtocol, string> = {
  iec104: 'IEC 104 status points and command points usually have different addresses (IOA). Check the command address in the device point list.',
  dnp3: 'DNP3 commands go to binary outputs (CROB) and analog outputs by index. The index of the output status point is usually the one to command.',
  modbus: 'Modbus writes coils (FC5/FC15) and holding registers (FC6/FC16).',
};

const DISCOVERY: Record<MasterProtocol, string> = {
  iec104: 'When it connects it asks for every value by itself (general interrogation), so all points appear without typing addresses.',
  dnp3: 'When it connects it asks for every value by itself (integrity poll) and then receives events, so all points appear without typing addresses.',
  modbus: 'Modbus cannot list its points: tell ScadaBench which ranges to read, or let Find registers discover them.',
};
export const ASK_AGAIN: Record<MasterProtocol, { label: string; hint: string }> = {
  iec104: { label: 'Ask for all values again', hint: 'Sends a general interrogation: the device answers with every value it has.' },
  dnp3: { label: 'Integrity poll now', hint: 'Asks the outstation for every current value (class 0, 1, 2 and 3).' },
  modbus: { label: 'Read now', hint: 'Reads every range in What to read once more.' },
};

export const MASTER_COPY = {
  intro: (protocol: MasterProtocol) => `ScadaBench connects as ${SIDES[protocol].us} to your ${SIDES[protocol].device}: a gateway, RTU, PLC or protection device. ${DISCOVERY[protocol]}`,
  host: 'Device address', hostHint: 'The IP address or name of the gateway, RTU or outstation, for example 192.168.1.20.',
  // Each protocol repeats "ask for everything" differently (change 0089: Modbus showed the 104 minutes hint).
  intervalHint: {
    iec104: 'Set the minutes to 0 to ask for all values only when connecting.',
    dnp3: 'Set the minutes to 0 to run the integrity poll only when connecting.',
    modbus: 'ScadaBench reads every range in What to read at this interval.',
  } as Record<MasterProtocol, string>,
  connect: 'Connect', disconnect: 'Disconnect', settingsTitle: 'Device connection',
  connecting: (host: string, port: unknown) => `Connecting to ${host}:${port}…`,
  connectingSlow: 'Still trying. The device has until the connect timeout (t0) to answer.',
  notBuilt: 'The master program is not built in this copy of ScadaBench.',
  emptyNotConnected: 'Points appear here by themselves after Connect.',
  states: { disconnected: 'Not connected', connected: 'Connected', lost: 'Link lost, trying again', failed: 'Stopped with a problem' },
  autoConnect: 'ScadaBench was connected when it closed, so it connects to the same device again by itself.',
  allowList: 'The device must accept connections from this PC: many gateways only answer known IP addresses.',
  points: (count: number) => count === 1 ? '1 point found' : `${count} points found`,
  noPoints: 'No points yet. Connect: ScadaBench asks the device for everything and lists what it answers.',
  noModbusPoints: 'No points yet. Modbus cannot list its points: add ranges or use Find registers in What to read.',
  pointsHint: 'Every point the device reported. All are watched; untick what you do not need. Name points so the list reads like your SCADA.',
  tickAll: 'Watch all', untickAll: 'Watch none', onlyWatched: 'Only watched', exportCsv: 'Export CSV', importLabels: 'Import names',
  labelPlaceholder: 'add a name', stale: 'Not connected: these are the last values received.',
  sendTo: 'Command…', commandTitle: 'Send a command to the device',
  value: 'Value', code: 'Control code', select: 'Select before operate', kind: 'Command type',
  doubleValues: { 1: '1 = OFF / open', 2: '2 = ON / close' } as Record<number, string>,
  confirmTitle: (name: string, address: number | undefined, host: string, word = 'address') =>
    `Send ${name}${address !== undefined ? ` to ${word} ${address}` : ''} on ${host}?`,
  realWarning: 'This goes to a REAL device. It can switch real equipment. Make sure it is safe and that people on site know.',
  send: 'Yes, send it', cancel: 'Cancel', openForm: 'New command…',
  noCommands: 'No command yet. Use New command… or the Command… button of a point.',
  commandsHint: 'Every command you sent: what was asked and how the device answered (confirmation, refusal, termination).',
  trafficHint: 'Decoded messages between ScadaBench and the device, newest at the bottom.',
  states_command: { sent: 'waiting…', confirmed: 'confirmed', refused: 'refused' },
};

export const TABLE_NAMES: Record<string, string> = {
  holding_registers: 'Holding registers (FC3)', input_registers: 'Input registers (FC4)',
  coils: 'Coils (FC1)', discrete_inputs: 'Discrete inputs (FC2)',
};
export const DATA_TYPE_NAMES: Record<string, string> = {
  uint16: 'uint16', int16: 'int16', uint32: 'uint32 (2 registers)', int32: 'int32 (2 registers)',
  float32: 'float32 (2 registers)', bool: 'on/off',
};
/** Find registers sends at most 10 requests per second. */
const duration = (requests: number) => requests < 600 ? `about ${Math.max(1, Math.round(requests / 10))} s`
  : `about ${Math.round(requests / 600)} min`;
export const RANGES_COPY = {
  tab: 'What to read',
  hint: 'Modbus cannot list its points, so ScadaBench reads the ranges below every poll interval. Use Find registers to discover which addresses answer.',
  empty: 'No ranges yet. Add one, or use Find registers.',
  add: 'Add range', save: 'Save ranges', table: 'Table', start: 'Start address', count: 'How many values', type: 'Read as',
  order: 'Word order', big: 'High word first', little: 'Low word first',
  findTitle: 'Find registers', find: 'Find registers', from: 'From address', to: 'To address',
  findHint: 'Reads only. ScadaBench reads blocks and narrows down where the device refuses, at most 10 requests per second.',
  preview: (fewest: number, most: number) =>
    `At least ${fewest} request${fewest === 1 ? '' : 's'} (${duration(fewest)}); up to ${most} (${duration(most)}) if the device refuses many addresses.`,
  running: (requests: number, address: number) => `Checking… ${requests} requests so far, at address ${address}`,
  found: (count: number) => count === 0 ? 'No address answered.' : `${count} answering range${count === 1 ? '' : 's'} found:`,
  addFound: 'Read these as', addFoundButton: 'Add to the read list',
  refused: (count: number) => `${count} range${count === 1 ? '' : 's'} refused by the device`,
};

export const TRAFFIC_WORDS = { rx: 'from the device', tx: 'to the device', info: 'note' };
