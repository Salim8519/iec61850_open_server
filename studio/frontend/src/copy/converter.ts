// Wording for the protocol converter role (ADR-0061): the device is read downstream as
// master, the same points are served upstream to SCADA as slave/outstation/server.
export const CONVERTER_COPY = {
  deviceTitle: 'Downstream: the device ScadaBench reads',
  scadaTitle: 'Upstream: your SCADA',
  intro: 'ScadaBench serves the watched device points to your SCADA. Connect to the device first, then start the upstream side. Untick device points you do not want to serve.',
  realCommands: 'Pass SCADA commands to the device',
  realCommandsOn: 'Commands from your SCADA on relayed points go on to the real device: binary outputs (DNP3), coils (Modbus) and your command routes.',
  realCommandsOff: 'The device is only read. SCADA gets a negative answer for every command.',
  start: 'Start serving', stop: 'Stop serving',
  startHint: 'Connect to the device, then press Start serving.',
  tabs: { device: 'Device points', served: 'Served to SCADA', routes: 'Command routes', scada: 'SCADA traffic',
    deviceTraffic: 'Device traffic', commands: 'Commands' },
  servedHint: 'What your SCADA sees: each watched device point with its upstream address. Addresses stay the same where the upstream protocol allows it.',
  servedEmpty: 'Nothing to serve yet: connect to the device and wait for its values.',
  routesHint: 'IEC 104 devices do not tell which addresses accept commands. Add a route for each command SCADA may send: the upstream address, and the device command and address it becomes.',
  routesEmpty: 'No command routes. DNP3 binary outputs and Modbus coils are relayed without a route.',
  addRoute: 'Add route', saveRoutes: 'Save routes',
  upstream: 'Upstream address', upstreamKind: 'Upstream command', deviceKind: 'Device command', deviceAddress: 'Device address',
  select: 'Select before operate',
};
export const UPSTREAM_KINDS: Record<string, string> = { command_double: 'Double command', command_single: 'Single command' };
export const DEVICE_KINDS: Record<string, string> = {
  double: 'IEC 104 double command (C_DC)', single: 'IEC 104 single command (C_SC)', binary_output: 'DNP3 CROB', coil: 'Modbus coil',
};
