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
  running: 'Running',
};
