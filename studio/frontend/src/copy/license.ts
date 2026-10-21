// Plain-language wording for licensing. Shown on the license screen and in Settings.
import type { LicenseStatus } from '../api/license';
import type { Status } from './status';
import { vmHostState } from './vmHost.ts';

export const LICENSE_COPY = {
  title: 'Activate ScadaBench',
  intro: 'ScadaBench needs a license on this computer. Try it free for 7 days, or enter the key you bought.',
  trialButton: 'Start 7-day free trial',
  trialHint: 'One free trial per computer. Needs internet to start; no payment details asked.',
  emailLabel: 'Your email address',
  emailHint: 'We send a 6-digit code. Your address is only used to allow one free trial per person.',
  sendCode: 'Send code',
  codeLabel: 'Code from the email',
  codeHint: 'Check your inbox (and spam folder). The code works for 15 minutes.',
  startWithEmail: 'Start free trial',
  checkOnlineButton: 'Check online now',
  checkOnlineHint: 'Confirms the date with the license server once. Your trial continues right away.',
  keyLabel: 'License key',
  keyPlaceholder: 'SUBS-XXXX-XXXX-XXXX-XXXX',
  keyHint: 'The key is in your purchase email. It works on one computer only and cannot be moved to another.',
  activateButton: 'Activate this computer',
  buyButton: 'Buy lifetime license',
  buyHint: 'Opens the secure purchase page in your browser. Your key appears right after payment.',
  optionsTitle: 'Server & offline options',
  machineLabel: 'Machine ID',
  machineHint: 'Identifies this computer. You only need it for offline activation or when contacting support.',
  copyMachine: 'Copy Machine ID',
  serverLabel: 'License server',
  serverHint: 'ScadaBench always asks its own license server. This address cannot be changed.',
  offlineTitle: 'Offline activation (computer without internet)',
  offlineSteps: [
    'Copy the activation code above, or the Machine ID if no code is shown (for example into a text file on a USB stick).',
    'On any computer with internet, open the offline activation page and enter your key and that code.',
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
  if (status.mode === 'vm-host') return vmHostState(status);
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
