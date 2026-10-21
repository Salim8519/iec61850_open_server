import type { IedRole } from '../api/types';

/** The three things ScadaBench can do with an IED, in one place (change 0142, owner-approved
 *  names): the full label, a short chip label and the one sentence shown under the choice. */
export const ROLE_COPY: Record<IedRole, { label: string; chip: string; sentence: string }> = {
  simulated: { label: 'Simulate', chip: 'Simulate',
    sentence: 'ScadaBench plays this IED; SCADA and other IEDs connect to it.' },
  real: { label: 'Monitor real device', chip: 'Monitor real',
    sentence: 'ScadaBench reads the real IED on your network and shows its live values on the SLD and through the gateway.' },
  off: { label: 'Skip', chip: 'Skip',
    sentence: 'Kept in the project (drawing and GOOSE stay), but not started when you press Run.' },
};

export const ROLE_ORDER: IedRole[] = ['simulated', 'real', 'off'];

/** "Simulate: … Monitor real device: … Skip: …" for the help tip next to the choice. */
export const ROLE_HELP = ROLE_ORDER.map(role => `${ROLE_COPY[role].label}: ${ROLE_COPY[role].sentence}`).join(' ');
