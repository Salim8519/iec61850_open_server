// Words of the full view of big lists (ADR-0095).
export const FULL_VIEW_COPY = {
  open: 'Full view',
  openHint: 'Show only this list, filling the window. Every button and filter stays. Esc goes back.',
  exit: 'Exit full view (Esc)',
  exitHint: 'Go back to the normal page',
  region: 'in full view',
};

/** Where the full view is, shown in its slim bar. */
export const FULL_PLACE = {
  gateway: 'SCADA gateway › Station gateway', master: 'SCADA gateway › SCADA master', converter: 'SCADA gateway › Protocol converter',
};
export const SNMP_FULL_PLACE = { simulate: 'SNMP devices › Simulate', read: 'SNMP devices › Read' };
export const EXPLORER_FULL_PLACE = 'IED Explorer';
export const gatewayFullDetail = (protocol: string, running: boolean) =>
  `${protocol === 'iec104' ? 'IEC 104' : protocol === 'modbus' ? 'Modbus TCP' : protocol === 'dnp3' ? 'DNP3' : protocol} · ${running ? 'running' : 'stopped'}`;
