// Wording of the SNMP devices tool (ADR-0092). Plain English for people new to SNMP.
import type { Behaviour, Level, PointType, SnmpRole } from '../api/snmp';

export const SNMP_ROLES: Record<SnmpRole, { label: string; sentence: string }> = {
  simulate: { label: 'Simulate devices',
    sentence: 'ScadaBench plays network devices (switches, GPS clocks) so your SCADA or network monitor can read them and receive their traps.' },
  read: { label: 'Read devices',
    sentence: 'ScadaBench reads a real or simulated SNMP device, shows its values with names, re-reads them and receives traps.' },
};

export const SNMP_COPY = {
  actAs: 'Act as',
  roleHint: 'Simulate devices: ScadaBench is the SNMP agent (the device). Read devices: ScadaBench is the SNMP manager (the reader). Both can run at the same time.',
  loading: 'Loading SNMP devices…',
  notAvailable: 'SNMP devices need the ScadaBench engine of version 0.1.17 or later. Restart ScadaBench to use them.',
  devices: 'Devices', addDevice: 'Add device', start: 'Start devices', stop: 'Stop devices',
  startHint: 'Each device answers on its own address, port 161, until you press Stop. They start again by themselves when ScadaBench reopens.',
  noDevices: 'No simulated devices yet',
  noDevicesHint: 'Press Add device and pick a Network switch or a GPS time server. It works at once: read it from Read devices, or point your SCADA or network monitor at its address.',
  pickDevice: 'Pick a device on the left to see its values and settings.',
  running: (count: number) => `${count} ${count === 1 ? 'device answers' : 'devices answer'} SNMP`,
  stopped: 'Devices are stopped',
  /** After Start, devices that could not open their address or port; the reason is on each device card. */
  notStarted: (failed: number, running: number) => running
    ? `${failed} ${failed === 1 ? 'device' : 'devices'} could not start. The reason is under ${failed === 1 ? 'it' : 'each one'} in the list.`
    : `${failed === 1 ? 'The device' : `${failed} devices`} could not start. The reason is under ${failed === 1 ? 'it' : 'each one'} in the list.`,
  states: { running: 'Answering', stopped: 'Stopped', failed: 'Problem' } as Record<string, string>,
  addMenu: { switch: 'Network switch (8 ports)', gps: 'GPS time server', empty: 'Empty device',
    walk: 'Copy of a device (snmpwalk file)…', csv: 'From a ScadaBench CSV file…' },
  importTitle: 'Import a device', importName: 'Name of the new device',
  importWalkHint: 'Save the output of "snmpwalk -v2c -c public <address> .1" (Net-SNMP) to a text file and pick it here. The device answers with exactly those values.',
  importProblems: (count: number) => `${count} ${count === 1 ? 'line was' : 'lines were'} skipped:`,
  remove: 'Delete device', removeConfirm: (name: string) => `Delete ${name}? Its settings and values are removed from the project.`,
  deviceMenu: 'More actions for this device',
  tabs: { values: 'Values', settings: 'Settings', traps: 'Traps', log: 'Requests' },
  tabLabel: (tab: string, text: string, count: number) => tab === 'values' ? `${text} (${count.toLocaleString('en-US')})` : text,
  restartNote: 'Running devices restart with the new settings.',
};

export const SETTINGS_COPY = {
  name: 'Name', address: 'Address (IP)', port: 'Port',
  addressHint: 'Each device has its own 127.x.x.x address that works on this PC only. To reach the devices from other PCs, put the project on a network in Settings › Network: each SNMP device gets its own free address on the same card as the IEDs.',
  localAddress: 'Address on this PC',
  networkNote: (address: string) => `On the network it answers at ${address}, the address Settings › Network gave it. It also keeps answering at its address on this PC.`,
  onNetwork: 'on the network', onNetworkLine: 'On the network',
  portHint: 'SNMP uses port 161. If Windows\' own SNMP service holds it, ScadaBench tells you; then pick 1161.',
  versions: 'SNMP versions it answers', versionsHint: 'v1 and v2c use a community name like a shared password. v3 uses a user name with real passwords and encryption.',
  readCommunity: 'Read community', writeCommunity: 'Write community',
  communityHint: 'Managers that use the read community can only read. The write community can also change values marked Writable.',
  users: 'v3 users', addUser: 'Add user', userName: 'User name', level: 'Security',
  auth: 'Authentication', authPassword: 'Authentication password', priv: 'Encryption', privPassword: 'Encryption password',
  passwordHint: 'At least 8 characters. Passwords are kept in the project file and never written to logs.',
  show: 'Show', hide: 'Hide', removeUser: 'Remove user',
  engineId: 'Engine ID', engineHint: 'The device\'s SNMPv3 identity (hex). Managers learn it by themselves; change it only to match a real device.',
  save: 'Save settings', saved: 'Saved',
};

export const LEVELS: Record<Level, string> = {
  noAuthNoPriv: 'No password',
  authNoPriv: 'Password only',
  authPriv: 'Password + encryption',
};
export const LEVEL_HINT = 'No password: anyone who knows the user name can read. Password only: every message is signed, so it cannot be changed or faked, but it is readable on the network. Password + encryption: signed and encrypted, the most secure.';
export const AUTH_NAMES: Record<string, string> = { md5: 'MD5', sha: 'SHA-1', sha224: 'SHA-224', sha256: 'SHA-256',
  sha384: 'SHA-384', sha512: 'SHA-512' };
export const PRIV_NAMES: Record<string, string> = { des: 'DES', aes128: 'AES-128', aes192: 'AES-192', aes256: 'AES-256',
  aes192c: 'AES-192 (Cisco key style)', aes256c: 'AES-256 (Cisco key style)' };
export const PRIV_HINT = 'AES-192 and AES-256 come in two key styles. If a Cisco device or Net-SNMP refuses the standard style, try the Cisco key style.';

export const TYPE_NAMES: Record<PointType, string> = { integer: 'Integer', octets: 'Text', oid: 'OID',
  ipaddress: 'IP address', counter32: 'Counter32', gauge32: 'Gauge32', timeticks: 'TimeTicks', counter64: 'Counter64',
  opaque: 'Opaque', null: 'Null' };
export const BEHAVIOURS: Record<Behaviour, string> = { fixed: 'Fixed', uptime: 'Counts up time', counter: 'Grows' };

export const POINTS_COPY = {
  search: 'Search name, OID or value', name: 'Label · name · OID',
  label: 'Label', labelPlaceholder: 'Add a label…',
  labelHint: 'Your own name for this value, such as "Uplink port 1". Saved with the project; managers never see it.',
  changeOid: 'Change OID', oidHint: 'Enter saves · Esc cancels · a MIB name works too', oid: 'OID', type: 'Type', value: 'Value', behaviour: 'Changes',
  writable: 'Writable', live: 'Live values', stored: 'Stored values',
  storedHint: 'Start the devices to see the values change. Until then you see the values saved in the project.',
  rate: 'per second', add: 'Add value', addTitle: 'Add a value of your own',
  addHint: 'Type an OID (1.3.6.1.4.1.…) or a name a MIB knows, such as IF-MIB::ifAlias.1.',
  addButton: 'Add to the device', remove: 'Remove', exportCsv: 'Export CSV', importCsv: 'Import CSV',
  writableHint: 'Writable values can be changed by managers that use the write community or a v3 user.',
  editHint: 'Click a value to change it. Managers see the new value at their next read.',
  none: 'No values match the filter.',
  more: 'More', moreLabel: 'More actions for values',
  labelOf: (name: string) => `Label of ${name}`, oidOf: (name: string) => `OID of ${name}`,
  valueOf: (name: string) => `Value of ${name}`, changesOf: (name: string) => `How ${name} changes`,
  helpText: 'Click a label, an OID or a value to change it; a pencil marks what you can edit. A label is your own name for a value, such as "Uplink port 1": it is saved with the project and managers never see it. An OID can also be a name a MIB knows. Values marked auto change by themselves; pick Fixed under Changes to type one. Managers see a new value at their next read.',
  select: 'Select', selectHint: 'Choose several values to delete',
  removeMany: (count: number) => `Delete ${count} values from this device? Managers no longer see them.`,
};

export const TRAPS_COPY = {
  targets: 'Send traps to', addTarget: 'Add trap receiver', host: 'Receiver address', port: 'Port',
  community: 'Community', user: 'v3 user', inform: 'Ask for a reply (inform)',
  informHint: 'An inform is a trap the receiver must acknowledge; ScadaBench repeats it until it does, up to 3 times.',
  targetsHint: '127.0.0.1 port 162 is Read devices › Traps on this PC. Add your SCADA or network monitor here.',
  event: 'Event', send: 'Send', sendHint: 'Sends the event as a trap (or inform) to every receiver above. The value changes first, like on the real device.',
  noTargets: 'Add a trap receiver first.', results: 'Result',
};

export const LOG_COPY = {
  empty: 'No requests yet. Read a device from Read devices, your SCADA or snmpget, and each request appears here.',
  time: 'Time', device: 'Device', from: 'From', what: 'Request', result: 'Result',
  held: (count: number) => ` · ${count} more not shown`,
  andMore: (count: number) => ` and ${count} more`,
};
