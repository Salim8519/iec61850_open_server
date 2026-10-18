// Filters of the Find IEDs results (change 0093): display only, on the scan the page already has.
import type { FoundDevice } from '../../../api/tools';
import type { Project } from '../../../api/types';
import { textMatch, type FilterSpec, type FilterValues } from '../../../ui/filterState.ts';

export const FIND_FILTER: FilterSpec = {
  search: 'Search address, name or vendor',
  selects: [{ key: 'vendor', label: 'Only one vendor', all: 'All vendors' }],
  toggles: [
    { key: 'answered', label: 'Answered', hint: 'Devices that told who they are (vendor, model, firmware).' },
    { key: 'iec', label: 'IEC 61850 only', hint: 'Devices that listed IEC 61850 logical devices.' },
    { key: 'unknown', label: 'Not in project', hint: 'Addresses that no IED of this project uses yet.' }],
};

/** Every address an IED of this project is known by: the SCD address, a typed real-device address, the lab address. */
export function knownAddresses(project: Pick<Project, 'ieds'>): string[] {
  return [...new Set(project.ieds.flatMap(ied => [ied.ip, ied.live?.ip, ied.lab_ip, ied.scl_address?.ip])
    .filter((ip): ip is string => Boolean(ip)))];
}

export function foundMatches(device: FoundDevice, values: FilterValues, known: Set<string>): boolean {
  if (values.answered && device.identified !== true) return false;
  if (values.iec && device.devices.length === 0) return false;
  if (values.unknown && known.has(device.ip)) return false;
  if (values.vendor && device.vendor !== values.vendor) return false;
  return textMatch(values.text, device.ip, device.vendor, device.model, device.revision, device.devices.join(' '));
}
