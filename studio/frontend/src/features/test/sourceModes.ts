// Generator modes in plain words. Values match the backend source specification.
export const MODES = [
  { value: 'manual', label: 'Set by hand', explain: 'The value only changes when you type a new one.' },
  { value: 'constant', label: 'Fixed value', explain: 'The value stays at the number you enter.' },
  { value: 'sine', label: 'Smooth wave', explain: 'The value swings up and down around the middle value.' },
  { value: 'ramp', label: 'Ramp', explain: 'The value climbs steadily and starts again each cycle.' },
  { value: 'step', label: 'Steps', explain: 'The value jumps between low and high each half cycle.' },
  { value: 'noise', label: 'Random noise', explain: 'The value moves randomly around the middle value.' },
  { value: 'csv', label: 'Play a recording (CSV)', explain: 'The value follows a recorded time/value file.' },
];

export const NUMERIC = (kind: string) => kind.startsWith('FLOAT') || kind.startsWith('INT');

export const modeLabel = (mode: string) => MODES.find(item => item.value === mode)?.label || mode;
