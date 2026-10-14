// Mode badges: text + icon (tone); never colour alone. Names follow IEC 61850-7-4.
import type { Status } from './status';

export const MODE_STATUS: Record<string, Status> = {
  on: { label: 'In service', tone: 'good', hint: 'Normal operation: the logical device works and acts on the process.' },
  'on-blocked': { label: 'Blocked', tone: 'warning', hint: 'Works, but its outputs to the process are blocked.' },
  test: { label: 'Test mode', tone: 'warning', hint: 'In test: it reacts to test messages and its outputs are marked as test.' },
  'test/blocked': { label: 'Test, blocked', tone: 'warning', hint: 'In test with outputs blocked: safe for testing without operating the plant.' },
  off: { label: 'Off', tone: 'off', hint: 'Switched off: it does not work and does not answer most services.' },
  simulation: { label: 'Simulation', tone: 'warning', hint: 'The IED listens to simulation GOOSE/SV from test equipment instead of the real messages.' },
  'not available': { label: 'Mode not available', tone: 'off', hint: 'This IED has no Mod/Beh data, so its mode cannot be shown.' },
};

export const MODE_CHOICES: { value: number; label: string }[] = [
  { value: 1, label: 'In service (on)' }, { value: 2, label: 'Blocked (on-blocked)' }, { value: 3, label: 'Test' },
  { value: 4, label: 'Test, blocked' }, { value: 5, label: 'Off' }];

export const modeStatus = (name: string): Status => MODE_STATUS[name] ?? {
  label: `Unknown mode ${name}`, tone: 'warning', hint: 'The device reports a value that is not a valid mode (1 to 5).' };

/** The one badge to show for a device: the worst mode, or simulation; nothing when in service. */
export function deviceBadge(summary?: { worst: string | null; simulation: boolean | null }): string | null {
  if (!summary) return null;
  if (summary.worst && summary.worst !== 'on') return summary.worst;
  return summary.simulation ? 'simulation' : null;
}
