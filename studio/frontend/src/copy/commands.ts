// Words of the IED Explorer command panel (ADR-0104). Plain English for people new to IEC 61850.
export const COMMAND_COPY = {
  higher: 'Higher', lower: 'Lower', valueLabel: 'Value to send', pickLabel: 'Value to send',
  model: (text: string) => `How it is operated: ${text}`,
  limits: (min: number, max: number) => `Allowed: ${min} to ${max}`,
  selectFirst: (label: string) => `Select ${label}…`,
  sendNow: (label: string) => `Send ${label}…`,
  selected: (label: string, seconds: number) => `Selected "${label}". Operate within ${seconds} s or cancel.`,
  operate: 'Operate', cancelSelect: 'Cancel selection',
  expired: 'The selection ran out before Operate. Press Select again.',
  confirmSelect: (label: string, reference: string, host: string) => `Select "${label}" on ${reference} at ${host}?`,
  confirmSelectHint: 'Select reserves the control on the device. Nothing moves until you press Operate.',
  yesSelect: (label: string) => `Yes, select ${label}`,
  optionsTitle: 'Command options',
  optionsHint: 'Most commands need none of these. Change them only when a test asks for it.',
  test: 'Test command (the device answers but does not switch)',
  interlock: 'Ask the device to check interlocking',
  synchro: 'Ask the device to check synchronism (synchrocheck)',
  originator: 'Sent from', originatorName: 'Sender name',
  originatorNameHint: 'Shown in the device log as who sent the command.',
};

/** Originator categories of IEC 61850-7-2 (orCat). */
export const ORIGINATORS: { value: number; label: string }[] = [
  { value: 1, label: 'Bay control' }, { value: 2, label: 'Station control' }, { value: 3, label: 'Remote control (SCADA)' },
  { value: 4, label: 'Automatic, bay' }, { value: 5, label: 'Automatic, station' }, { value: 6, label: 'Automatic, remote' },
  { value: 7, label: 'Maintenance' }, { value: 8, label: 'Process' },
];
