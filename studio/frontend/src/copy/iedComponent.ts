import type { IedRole } from '../api/types';

/** Words of IED components, GOOSE links and "Add all GOOSE IEDs" (ADR-0112). Plain English for
 * beginners; every state is words first, the border repeats it. */
const many = (count: number, one: string, more = `${one}s`) => `${count} ${count === 1 ? one : more}`;

export const IED_COPY = {
  chooseIed: 'Choose which IED this is in its sheet on the right.',
  cardHint: 'One whole IED. GOOSE lines start, pass and end here. Click for its messages.',
  state(role: IedRole, locked: boolean, running: boolean): { text: string; tone: 'running' | 'stopped' | 'real' | 'off' } {
    if (role === 'off') return { text: 'Skipped in runs', tone: 'off' };
    if (role === 'real') return { text: locked ? running ? 'Real device · watched' : 'Real device · not in this run'
      : 'Real device · watch only', tone: 'real' };
    return { text: running ? 'Simulated · running' : locked ? 'Simulated · not running' : 'Simulated · stopped',
      tone: running ? 'running' : 'stopped' };
  },
  goose(sends: number, gets: number): string {
    if (!sends && !gets) return 'No GOOSE';
    if (!sends) return `Gets ${many(gets, 'message')}`;
    return `Sends ${many(sends, 'message')}${gets ? ` · gets ${gets}` : ''}`;
  },
};

export const IED_SHEET_COPY = {
  pickTitle: 'Which IED is this?',
  pickHint: 'One component per IED. GOOSE lines on the drawing go from IED to IED through these components.',
  pickNone: 'Choose an IED',
  taken: (name: string) => `${name} (already on the drawing)`,
  lockedPick: 'Stop the run to choose another IED.',
  sends: 'Messages it sends',
  sendsNone: 'This IED sends no GOOSE in its files.',
  receives: 'Messages it receives',
  receivesNone: 'This IED receives no GOOSE in its files.',
  to: (names: string[]) => names.length ? `To ${names.join(', ')}` : 'No IED in this project receives it',
  incomplete: 'The values of this message are not listed: import this IED again to list them.',
  watchOnly: 'Watch only: ScadaBench never sends GOOSE for a real device.',
  sendHint: 'Change a value to send it: the IED raises the message\'s state number once and every receiver gets it.',
  loading: 'Reading its GOOSE messages',
  member: (position: number, text: string) => `${position}. ${text}`,
  notSendable: 'Not changed from here',
  /** A received message's link state in words (GET /goose/links status). */
  receiveStatus: (status: string): string => ({ declared: 'set up in the files', remote: 'sender not in this project',
    disabled_in_file: 'switched off in the sender file', no_dataset: 'the sender has no dataset for it',
    no_address: 'the sender has no GOOSE address', not_goose: 'not a GOOSE message', not_sendable: 'the sender cannot send it',
  } as Record<string, string>)[status] ?? status,
};

export const GOOSE_LINK_COPY = {
  title: 'GOOSE',
  hint: 'Optional. Send one GOOSE message of a simulated IED from this equipment. Its IEC 61850 connections above stay as they are.',
  add: 'Send GOOSE from this equipment',
  remove: 'Remove GOOSE',
  ied: 'IED',
  message: 'Message',
  value: 'Value',
  mode: 'What it sends',
  positionValue: 'Position or on/off value',
  send: (what: string) => `Send ${what}`,
  numberLabel: 'Value to send',
  openValue: 'Open value',
  closedValue: 'Closed value',
  one: 'One value (position or on/off)',
  contacts: 'Open and closed as two on/off values',
  chooseIed: 'Choose an IED that sends GOOSE',
  chooseMessage: 'Choose a message',
  chooseValue: 'Choose a value',
  noPublishers: 'No simulated IED in this project sends GOOSE. Import an IED whose file has GOOSE messages.',
  summary: (ied: string, message: string) => `Sends ${message} of ${ied}`,
  lockedEdit: 'Stop the run to change which GOOSE this equipment sends.',
  stoppedSend: 'Press Run to send this GOOSE.',
  sent: (words: string) => `Sent: ${words}. Watch the line on the drawing.`,
};

export const GOOSE_IEDS_COPY = {
  addAll: 'Add all GOOSE IEDs',
  addAllHint: 'One click: an IED component for every IED that sends or receives GOOSE, in a row above the drawing. One Undo step.',
  added: (count: number) => count === 1 ? 'Added 1 IED component next to the others. Undo removes it.'
    : `Added ${many(count, 'IED component')} above the drawing. Undo removes them all.`,
  none: 'Every IED that sends or receives GOOSE is already on the drawing.',
  missingButton: (count: number) => `${many(count, 'IED')} not on the drawing`,
  missingIntro: 'These IEDs send or receive GOOSE but have no IED component, so their lines cannot be drawn.',
  add: 'Add',
  locked: 'Stop the run to change the drawing.',
  addHint: (name: string) => `Add an IED component for ${name} next to the others`,
  alsoChip: (names: string[]) => `also ${names.length > 2 ? `${names.slice(0, 2).join(', ')} +${names.length - 2}` : names.join(', ')} · not on drawing`,
  alsoTitle: (names: string[]) => `Also sent to ${names.join(', ')}. Add their IED components to see these lines.`,
};

/** The tag on point panel rows whose value is part of a GOOSE message (ADR-0112). */
export const GOOSE_TAG = {
  text: 'GOOSE',
  title: (count: number) => `This value is sent in ${many(count, 'GOOSE message')}. Changing it here sends GOOSE, ` +
    'and with the GOOSE layer on its line starts at this panel.',
};
