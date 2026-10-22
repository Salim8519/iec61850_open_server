// Wording of SNMP devices › Read devices and the MIB library (ADR-0092).

export const READ_COPY = {
  connectTitle: 'Device to read', address: 'Address (IP)', port: 'Port', version: 'SNMP version',
  community: 'Community', communityHint: 'Most devices use public for reading. Ask the owner of the device if public does not answer.',
  user: 'v3 user', connect: 'Connect and read everything', disconnect: 'Disconnect', reconnect: 'Connect again',
  connectHint: 'ScadaBench reads every value of the device once (a walk), shows the important ones on top and then re-reads them by itself.',
  timeout: 'Wait for an answer', retries: 'Tries',
  every: 'Re-read every', everyHint: 'How often the values on top and your watched values are read again. The full walk runs only when you press Walk again.',
  walkAgain: 'Walk again', saveTwin: 'Save as simulated device', twinName: (name: string) => `Copy of ${name}`,
  twinSaved: (name: string, count: number) => `${name} was added to Simulate devices with ${count} values.`,
  states: { disconnected: 'Not connected', connecting: 'Connecting…', walking: 'Reading every value…',
    connected: 'Connected', failed: 'Problem' } as Record<string, string>,
  walking: (count: number) => count ? `Reading every value… ${count.toLocaleString()} so far` : 'Reading every value…',
  values: (count: number, polled: number) => `${count.toLocaleString()} values · ${polled} kept up to date`,
  emptyTitle: 'Read any SNMP device',
  emptyHint: 'Type its address on the left and press Connect. To try it now, start a simulated device in Simulate devices: its address is already filled in.',
  search: 'Search name, OID, value, meaning', keyOnly: 'Important only', watchedOnly: 'Watched only',
  keyHint: 'The device\'s name and description, port states, sensors, battery and clock status.',
  important: 'Important value',
  watch: 'Watch', unwatch: 'Stop watching', watchHint: 'Watched values are re-read with the important ones.',
  set: 'Change', setTitle: (name: string) => `Change ${name} on the device?`,
  setHint: 'This writes the new value into the device, like a network management system would. Many devices refuse values they do not allow.',
  setSend: 'Yes, change it', cancel: 'Cancel', newValue: 'New value',
  system: 'Device', none: 'No values match the filter.',
};

/** A refused SET with the reason a beginner can act on (0178): v1/v2c devices usually take writes only with
 *  the write community, which is not the read community public. */
export function refusedWrite(message: string): string {
  if (!/no access|not writable|not authorized|read-only/i.test(message)) return message;
  return `${message}. Changing values usually needs the write community (often private), or a v3 user that may write. `
    + 'Connect again with it, then try the change again.';
}

export const TRAP_RECEIVER_COPY = {
  title: 'Trap receiver', listen: 'Receive traps', stop: 'Stop receiving', where: 'Listen on', port: 'Port',
  thisPc: 'This PC only (127.0.0.1)', allCards: 'All network cards (0.0.0.0)',
  allCardsHint: 'Real devices send traps over the network. Windows may ask once whether ScadaBench SNMP may use the network: choose Allow on private networks.',
  usersHint: 'v3 traps are accepted from these users (the same user name and passwords as on the device).',
  empty: 'No traps yet. Start receiving, then send an event from Simulate devices › Traps, or let a real device send one.',
  inform: 'inform (acknowledged)', trap: 'trap', from: 'From', what: 'Event', values: 'Values',
  problem: 'Not accepted',
};

export const MIB_COPY = {
  title: 'MIB library', hint: 'MIBs give values their names and meanings, for example ifOperStatus = up. Standard MIBs are built in; add your device vendor\'s MIB files here. They are kept on this PC for every project.',
  importButton: 'Add MIB files…', remove: 'Remove', restore: 'Bring back removed built-in MIBs',
  builtin: 'Built in', imported: 'Added by you', names: (count: number) => `${count.toLocaleString()} ${count === 1 ? 'name' : 'names'}`,
  missing: (names: string[]) => `Needs ${names.join(', ')}: add those MIB files too.`,
  added: (names: string[]) => `Added ${names.join(', ')}.`, search: 'Search MIBs',
};
