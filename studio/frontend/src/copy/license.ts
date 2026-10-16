// Plain-language wording for licensing. Shown on the license screen and in Settings.
import type { LicenseStatus } from '../api/license';
import type { Status } from './status';

export const LICENSE_COPY = {
  title: 'Activate ScadaBench',
  intro: 'ScadaBench needs a license on this computer. Try it free for 7 days, or enter the key you bought.',
  trialButton: 'Start 7-day free trial',
  trialHint: 'One free trial per computer. Needs internet once; no payment details asked.',
  keyLabel: 'License key',
  keyPlaceholder: 'SUBS-XXXX-XXXX-XXXX-XXXX',
  keyHint: 'The key is in your purchase email. It works on one computer.',
  activateButton: 'Activate this computer',
  buyButton: 'Buy lifetime license',
  buyHint: 'Opens the secure purchase page in your browser. Your key appears right after payment.',
  moveQuestion: 'This key is active on another computer. Move it to this computer?',
  moveHint: 'The other computer stops working when it next goes online. Moves are limited per year.',
  moveButton: 'Move the key here',
  optionsTitle: 'Server & offline options',
  machineLabel: 'Machine ID',
  machineHint: 'Identifies this computer. You only need it for offline activation or when contacting support.',
  copyMachine: 'Copy Machine ID',
  serverLabel: 'License server address',
  serverHint: 'Only change this if the seller gave you a new address.',
  offlineTitle: 'Offline activation (computer without internet)',
  offlineSteps: [
    'Copy the Machine ID above (for example into a text file on a USB stick).',
    'On any computer with internet, open the offline activation page and enter your key and the Machine ID.',
    'Bring the downloaded license.cert back to this computer and load it below.',
  ],
  loadFile: 'Load license file',
  pasteLabel: 'Or paste the license file text',
  offlineButton: 'Activate offline',
  safeData: 'Your projects are safe: they stay in the data folder and open again as soon as ScadaBench is activated.',
};

export function licenseState(status: LicenseStatus): Status {
  if (status.mode === 'development') {
    return { label: 'Development mode', tone: 'info', hint: 'Licensing is off while developing. Packaged builds check the license.' };
  }
  if (status.mode === 'lifetime') {
    return { label: 'Lifetime license', tone: 'good', hint: 'This computer is licensed. No further action is needed.' };
  }
  if (status.mode === 'trial') {
    const days = status.days_remaining ?? 0;
    return { label: `Free trial: ${days} day${days === 1 ? '' : 's'} left`,
      tone: days <= 2 ? 'warning' : 'info', hint: 'Buy a lifetime license before the trial ends to keep working without a break.' };
  }
  return { label: 'Not activated', tone: 'problem', hint: status.reason || LICENSE_COPY.intro };
}
