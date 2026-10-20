// Wording for the Tools place. Each tool card lists only a tool that is built and usable;
// planned tools are specified in docs/specs/proposed/ and appear here once delivered.
export type ToolId = 'explorer' | 'find' | 'gateway' | 'snmp' | 'traffic';

export type ToolCard = { id: ToolId; name: string; line: string; hint: string };

export const TOOLS: ToolCard[] = [
  { id: 'explorer', name: 'IED Explorer',
    line: 'Connect to any IED by address, browse its data, watch live values and send commands.',
    hint: 'An IEC 61850 client. Type an IP address: Explorer opens the data model and watches the key points by itself. Add any value to the watch list, and operate a breaker or switch after one confirmation.' },
  { id: 'find', name: 'Find IEDs',
    line: 'Scan a network for IEC 61850 devices and see who they are.',
    hint: 'Checks every address of a network for port 102, then asks each device for its vendor, model and firmware. Open any result straight in IED Explorer.' },
  { id: 'gateway', name: 'SCADA gateway & master',
    line: 'Be the station gateway for your SCADA, or be the SCADA master for your gateway or RTU.',
    hint: 'Station gateway: ScadaBench serves your IEDs to SCADA over IEC 104, Modbus TCP or DNP3. SCADA master: it connects to a gateway or RTU, finds every point by itself and sends commands after a confirmation.' },
  { id: 'snmp', name: 'SNMP devices',
    line: 'Simulate network devices such as switches and GPS clocks, or read any SNMP device.',
    hint: 'SNMP is how switches, GPS clocks and UPS units report their health. Simulate devices plays them in v1, v2c and v3 and sends their traps. Read devices reads any device, names its values from MIBs and receives traps.' },
  { id: 'traffic', name: 'GOOSE & Sampled Values traffic',
    line: 'See which GOOSE and Sampled Values messages are really on the network.',
    hint: 'Checks every GOOSE block and Sampled Values stream of your project against what really arrives. GOOSE needs nothing extra; Sampled Values need the free Npcap driver and a lab network.' },
];

export const TOOLS_COPY = {
  intro: 'Handy helpers for testing and commissioning. Pick a tool; All tools returns to this list.',
  back: 'All tools',
  whatIsThis: 'What is this?',
  settingsLink: 'GOOSE and Sampled Values traffic now has its own place under Tools.',
  openTraffic: 'Open traffic in Tools',
};

export const EXPLORER_COPY = {
  intro: 'Type the address of an IED (or a simulated one while Run is on) and press Connect. Explorer opens the key points by itself and keeps them updated. It never takes a report block, and it sends a command only after you confirm it.',
  address: 'IED address', port: 'Port', connect: 'Connect', disconnect: 'Disconnect',
  readOnly: 'Connected',
  treeTitle: 'Data model', treeHint: 'Open a logical device, then a logical node. Each level loads only when you open it, so even large IEDs open quickly.',
  filter: 'Filter logical nodes', pickNode: 'Pick a logical node on the left to see its data.',
  objectsHint: 'Open a data object to read its values. Add any value to the watch list to keep it updated.',
  datasets: 'Datasets', reports: 'Report blocks',
  reportsHint: 'Report blocks are listed only. Explorer never enables or reserves them, so SCADA and gateways keep theirs.',
  watchTitle: 'Watch list', watchEmpty: 'Nothing watched yet. Press + next to any value to add it here.',
  watchHint: 'Values refresh at the chosen interval while Watch is on. Export saves them as a CSV file.',
  start: 'Start watching', stop: 'Stop watching', readOnce: 'Read now', exportCsv: 'Export CSV', clear: 'Clear', clearWatch: 'Clear the watch list',
  every: 'Refresh every', everyShort: 'Every',
  noMatch: 'Nothing matches the filter. Press Clear to see everything.',
  notConnected: 'Connect to an IED to see its data model here',
  command: 'Command:', cancel: 'Cancel', nowReads: 'The device now reads:',
  confirmTitle: (label: string, reference: string, host: string) => `Send "${label}" to ${reference} on ${host}?`,
  confirmHint: 'This operates the device for real, using its own control model (select before operate when it asks for it). Make sure it is safe to switch.',
  send: (label: string) => `Yes, send ${label}`,
};

export const FIND_COPY = {
  intro: 'Pick the network adapter the devices are plugged into. Find IEDs checks each address for port 102 (IEC 61850) and asks every device that answers who it is. It sends nothing else.',
  adapter: 'Network adapter', range: 'Addresses to check', port: 'Port',
  rangeHint: 'A network such as 192.168.1.0/24, one address, or a range such as 192.168.1.20-40. At most 256 addresses.',
  start: 'Find IEDs', stop: 'Stop', again: 'Scan again',
  willCheck: (count: number, first: string, last: string, port: number) =>
    count === 1 ? `Will check ${first} on port ${port}.` : `Will check ${count} addresses, ${first} to ${last}, on port ${port}.`,
  addresses: (count: number) => count === 1 ? '1 address' : `${count} addresses`,
  emptyTitle: 'Found devices appear here',
  emptyHint: 'Press Find IEDs. Every device that answers on the port is listed with its address, vendor, model, firmware and logical devices, ready to open in IED Explorer.',
  inProject: 'Already in this project',
  noMatch: 'No device matches the filter. Press Clear to see every result.',
  open: 'Open in Explorer', none: 'No device answered. Check the cable, the adapter and that the devices are switched on.',
  addHint: 'To test with one of these devices in a project, import its CID/SCD in Devices and choose "Connect to real device".',
  thisPc: 'This PC: simulated IEDs (127.50.0.x)',
  loopbackHint: 'Simulated IEDs answer on 127.50.0.x addresses while Run is on, so you can practise here without any cable.',
};
