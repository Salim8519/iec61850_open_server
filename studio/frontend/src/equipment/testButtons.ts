import type { EquipmentData, Ied, SignalBinding } from '../api/types';
import { relayBindings, stateBindings } from './stateBindings.ts';

/** Which test buttons an equipment shows (ADR-0067). Display only: the backend checks every
 * request again and writes the point through the sample writer; the browser decides nothing. */
export type TestButton = { action: string; label: string; row?: string; tone?: 'primary' | 'danger' };
export type TestAvailability = { buttons: TestButton[]; reason: '' | 'real' | 'not_running' | 'unbound' };

const SWITCH: TestButton[] = [{ action: 'open', label: 'Open' }, { action: 'closed', label: 'Closed', tone: 'primary' },
  { action: 'moving', label: 'Moving' }, { action: 'bad', label: 'Bad state', tone: 'danger' }];

export function testButtons(data: EquipmentData): TestButton[] {
  const mapping = data.state_mapping;
  if (data.kind === 'RELAY') return (data.relay_statuses || []).flatMap(entry => [
    { action: 'on', label: `${entry.label} on`, row: entry.id }, { action: 'off', label: `${entry.label} off`, row: entry.id }]);
  if (!mapping) return [];
  switch (data.kind) {
    case 'CB': case 'DS': case 'ES': return mapping.mode === 'single_to_dbpos' ? SWITCH.slice(0, 2) : SWITCH;
    case 'TRIP_RELAY': return [{ action: 'trip', label: 'Trip', tone: 'danger' }, { action: 'reset', label: 'Reset' }];
    case 'CT': case 'VT': return [{ action: 'set', label: 'Set value', tone: 'primary' }, { action: 'bad_quality', label: 'Bad quality', tone: 'danger' }];
    case 'SOURCE': case 'INFEED': return [{ action: 'live', label: 'Live', tone: 'primary' }, { action: 'dead', label: 'Dead' }];
    default: return [];
  }
}

/** Buttons only while running and only when every bound IED is simulated and running. */
export function testAvailability(data: EquipmentData, ieds: Ied[], running: Set<string>): TestAvailability {
  const buttons = testButtons(data);
  if (!buttons.length) return { buttons: [], reason: 'unbound' };
  const bound: SignalBinding[] = [...stateBindings(data.state_mapping), ...relayBindings(data)];
  const targets = bound.map(item => ieds.find(ied => ied.id === item.ied_id)).filter(Boolean) as Ied[];
  if (targets.some(ied => ied.role === 'real')) return { buttons: [], reason: 'real' };
  if (!targets.length || targets.some(ied => !running.has(ied.id))) return { buttons: [], reason: 'not_running' };
  return { buttons, reason: '' };
}

export const TEST_COPY = {
  title: 'Test',
  explain: (name: string) => `Sets the position in the simulated IED as if the ${name.toLowerCase()} moved by itself. No command is sent.`,
  explainOther: 'Sets the value in the simulated IED as if the plant changed by itself. No command is sent.',
  real: 'Reads a real IED: test buttons never change it. Use its command link to operate it.',
  notRunning: 'IED not running. Run this IED from Devices (or press Run) to use the test buttons.',
  commandHint: 'Add a command link to operate it like SCADA.',
  valueLabel: 'New value',
};
