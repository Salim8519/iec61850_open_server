import type { EquipmentData, EquipmentKind } from '../api/types';

/** Library category; describes the equipment family, never energized or protection state. */
export type Category = 'switchgear' | 'instrument' | 'source' | 'protection' | 'control' | 'drawing' | 'legacy';

type KindInfo = { name: string; code: string; category: Category; state: string };

const INFO: Record<EquipmentKind, KindInfo> = {
  CB: { name: 'Circuit breaker', code: 'XCBR', category: 'switchgear', state: 'Position' },
  DS: { name: 'Disconnector', code: 'XSWI', category: 'switchgear', state: 'Position' },
  ES: { name: 'Earth switch', code: 'XSWI', category: 'switchgear', state: 'Position' },
  CT: { name: 'Current transformer', code: 'TCTR', category: 'instrument', state: 'Measured value' },
  VT: { name: 'Voltage transformer', code: 'TVTR', category: 'instrument', state: 'Measured value' },
  SOURCE: { name: 'Analog source', code: 'MX', category: 'source', state: 'Measurement point' },
  RELAY: { name: 'Protection relay', code: 'Status panel', category: 'protection', state: 'Signals' },
  TRIP_RELAY: { name: 'Trip relay', code: 'SPC', category: 'protection', state: 'Trip state' },
  PUSH_BUTTON: { name: 'Push button', code: 'Boolean / SPC', category: 'control', state: 'Button state' },
  JUNCTION: { name: 'Junction', code: 'Drawing point', category: 'drawing', state: '' },
  TRANSFORMER: { name: 'Power transformer', code: 'Legacy', category: 'legacy', state: 'State' },
  LOAD: { name: 'Load', code: 'Legacy', category: 'legacy', state: 'State' },
  BUSBAR: { name: 'Busbar', code: 'Legacy', category: 'legacy', state: 'State' },
};

export function kindInfo(kind: EquipmentKind): KindInfo {
  return INFO[kind];
}

export type SetupItem = { label: string; done: boolean };

/** What an object needs before it can show live state or accept operation. */
export function setupItems(data: EquipmentData): SetupItem[] {
  const state = { label: kindInfo(data.kind).state, done: !!data.state_mapping };
  const command = { label: 'Command', done: !!data.control_binding };
  switch (data.kind) {
    case 'CB': case 'DS': case 'ES': case 'TRIP_RELAY': return [state, command];
    case 'CT': case 'VT': case 'SOURCE': return [state];
    case 'PUSH_BUTTON': return data.push_mode === 'spc_momentary' ? [command] : [state];
    case 'RELAY': return [{ label: 'At least one signal', done: !!data.relay_statuses?.length }];
    default: return [];
  }
}

/** Canvas state tone: unmapped, mapped but no value yet, or showing a live value. */
export function stateTone(linked: boolean, value: unknown): 'unmapped' | 'awaiting' | 'live' {
  if (!linked) return 'unmapped';
  return value === undefined || value === null ? 'awaiting' : 'live';
}
