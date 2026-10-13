import type { EquipmentData } from '../api/types';

export type SwitchPosition = 'Open' | 'Closed' | 'Intermediate' | 'Invalid' | 'Awaiting value' | 'Mapping required';
export function switchPosition(data: EquipmentData, value: unknown): SwitchPosition {
  if (!data.state_mapping) return 'Mapping required';
  if (value === undefined || value === null) return 'Awaiting value';
  return value === 2 ? 'Closed' : value === 1 ? 'Open' : value === 0 ? 'Intermediate' : 'Invalid';
}

export function tripRelayState(mapped: boolean, value: unknown, operatedValue = false) {
  if (!mapped) return 'Not mapped';
  if (value === undefined || value === null) return 'Awaiting state';
  if (typeof value !== 'boolean') return 'Invalid';
  return value === operatedValue ? 'Operated' : 'Reset';
}
