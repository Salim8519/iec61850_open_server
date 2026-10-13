import type { Status } from './status';

// Plain meaning of every comparison result between the SCD and the real device.
export const COMPARE: Record<string, Status> = {
  match: { label: 'Same in both', tone: 'good', hint: 'The device and the SCD agree on this point.' },
  missing_on_device: { label: 'Missing on device', tone: 'problem', hint: 'The SCD has it but the device does not. The device probably runs another configuration.' },
  extra_on_device: { label: 'Only on device', tone: 'info', hint: 'The device has it but the SCD does not. Usually harmless; the SCD may be older.' },
  type_mismatch: { label: 'Different type', tone: 'warning', hint: 'Both have it with different data types, so clients may read it wrongly.' },
  dataset_different: { label: 'Dataset differs', tone: 'warning', hint: 'The dataset exists on both but contains different points.' },
};
