import type { CheckProgress } from '../../api/network';
import type { Step } from '../../ui/StepList';

/** The steps shown before the first progress answer arrives (or when the engine has no progress
 *  route): the same four steps, the first one running. */
const FIRST: Step[] = [
  { key: 'cards', label: 'Read the network card', state: 'running' },
  { key: 'addresses', label: 'Check each address on the network', state: 'waiting' },
  { key: 'firewall', label: 'Read Windows Firewall rules', state: 'waiting' },
  { key: 'plan', label: 'Prepare the exact Windows changes', state: 'waiting' },
];

/** The step list for the address check: the engine's steps, with "12 of 30 addresses" on the
 *  address step while answers arrive. */
export function checkSteps(progress: CheckProgress | null): Step[] {
  if (!progress?.running) return FIRST;
  return progress.steps.map(step => step.key === 'addresses'
    ? { ...step, done: progress.answered, total: progress.asked, unit: progress.asked === 1 ? 'address' : 'addresses' }
    : step);
}

/** Words for one row while its address is checked; icons and words, never colour alone. */
export const ROW_STATE: Record<CheckProgress['rows'][number]['state'], string> = {
  checking: 'Checking…',
  free: 'Free: no other device answered',
  in_use: 'Taken: another device answers here',
  not_checked: 'Checked by Windows when it is added',
};
