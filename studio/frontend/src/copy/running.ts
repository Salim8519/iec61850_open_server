/** Words for the running dots on the sidebar and the tool cards. */
const plural = (count: number, word: string) => `${count} ${word}${count === 1 ? '' : 's'}`;

export const PROTOCOL_SHORT: Record<string, string> = { iec104: 'IEC 104', modbus: 'Modbus TCP', dnp3: 'DNP3' };

export const RUNNING_COPY = {
  ieds: (count: number) => `${plural(count, 'IED')} running`,
  iedsPaused: (count: number) => `${plural(count, 'IED')} running · automation paused`,
  diagram: 'Live diagram: IEDs are running',
  sequences: (count: number) => `${plural(count, 'sequence')} running`,
  gateway: (protocol: string, port: number | null, clients: number) =>
    `Gateway: ${PROTOCOL_SHORT[protocol] ?? protocol}${port ? ` on port ${port}` : ''} · ${plural(clients, 'SCADA connection')}`,
  gatewayFailed: 'Gateway stopped with a problem',
  master: (protocol: string, host: string, points: number, lost: boolean) =>
    `SCADA master: ${PROTOCOL_SHORT[protocol] ?? protocol} to ${host}${lost ? ' · link lost, retrying' : ''} · ${plural(points, 'point')}`,
  explorer: (host: string, watched: number, polling: boolean) =>
    `IED Explorer: connected to ${host}${polling ? ` · watching ${watched}` : ''}`,
  scan: (probed: number, total: number) => `Find IEDs: scanning ${probed} of ${total}`,
  snmpDevices: (count: number) => `SNMP: ${plural(count, 'simulated device')} answering`,
  snmpReading: (target: string) => `SNMP: reading ${target.split(' ')[0]}`,
  snmpTraps: 'SNMP: receiving traps',
  running: 'Running',
};

/** Top bar run state: "Running · 2 IEDs" when all run, "Running 1 of 2 IEDs" when only some do (ADR-0067). */
export function runStateWords(inRun: number, notOff: number, paused: boolean): string {
  if (paused) return 'Running · automation paused';
  return inRun >= notOff ? `Running · ${plural(inRun, 'IED')}` : `Running ${inRun} of ${plural(notOff, 'IED')}`;
}

export const RUN_ONE_COPY = {
  simulated: { start: 'Run this IED', stop: 'Stop this IED',
    startHint: 'Start only this simulated IED: its MMS server, reports, polling answers, GOOSE and all its points. Other IEDs stay as they are.',
    stopHint: 'Stop only this simulated IED. Other running IEDs keep going.' },
  real: { start: 'Connect', stop: 'Disconnect',
    startHint: 'Read this real device as a client: free report blocks, else polling. Nothing is sent unless you operate.',
    stopHint: 'Disconnect from this real device and give back every report block ScadaBench used.' },
};
