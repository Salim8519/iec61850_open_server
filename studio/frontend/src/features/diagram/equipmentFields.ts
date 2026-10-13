import type { EquipmentData, EquipmentKind } from '../../api/types';

// Which settings a beginner sees first and which wait under "Advanced" (UI-04).
export type SheetField = 'name' | 'start_position' | 'push_mode' | 'push_value' | 'push_rest_value'
  | 'push_reset_ms' | 'trip_command_polarity' | 'trip_state_polarity' | 'colors';

const SWITCHES: EquipmentKind[] = ['CB', 'DS', 'ES'];

export function basicFields(kind: EquipmentKind): SheetField[] {
  return SWITCHES.includes(kind) ? ['name', 'start_position'] : ['name'];
}

export function advancedFields(data: Pick<EquipmentData, 'kind' | 'push_mode'>): SheetField[] {
  if (SWITCHES.includes(data.kind)) return ['colors'];
  if (data.kind === 'TRIP_RELAY') return ['trip_command_polarity', 'trip_state_polarity', 'colors'];
  if (data.kind === 'PUSH_BUTTON') {
    const mode = data.push_mode || 'latching_status';
    return ['push_mode', ...(mode === 'spc_momentary' ? ['push_value' as const] : []),
      ...(mode === 'self_resetting' ? ['push_rest_value' as const, 'push_reset_ms' as const] : []), 'colors'];
  }
  return [];
}

/** Trip relay polarity in one plain sentence. */
export function polaritySentence(data: Pick<EquipmentData, 'trip_command_polarity' | 'trip_state_polarity'>) {
  const operate = data.trip_command_polarity === 'operate_true' ? 1 : 0;
  const operated = data.trip_state_polarity === 'operated_false' ? 0 : 1;
  return `Operate sends ${operate}, Reset sends ${1 - operate}. It shows Operated when the state is ${operated}.`;
}

export const PUSH_MODES = [
  { value: 'latching_status', label: 'Toggle on and off (default)' },
  { value: 'self_resetting', label: 'Press, then release by itself' },
  { value: 'spc_momentary', label: 'Send a control command (SPC)' },
] as const;
