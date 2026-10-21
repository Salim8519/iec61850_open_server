// Plain-language wording for sharing a lifetime license with virtual machines (ADR-0099).
import type { LicenseStatus } from '../api/license.ts';
import type { Status } from './status.ts';

export const VM_HOST_COPY = {
  title: 'Virtual machines',
  switchLabel: 'Let my virtual machines use this license',
  intro: 'Do you run ScadaBench inside a virtual machine (VM) on this PC, for example in Hyper-V, VMware or '
    + 'VirtualBox? Turn this on and your VMs can use this license without an internet connection and '
    + 'without a second key.',
  how: 'How it works: a small helper program starts when you sign in to Windows. A VM on this PC asks it '
    + 'over the virtual network. The helper only listens on the virtual network of your VMs, never on your '
    + 'office or internet network, and it answers only while this PC is licensed. Turn the switch off and the '
    + 'helper stops by itself.',
  keyLabel: 'Virtual machine key',
  keyHint: 'A short form of the key this PC uses to confirm its VMs. It is only shown here so you can compare it.',
  reading: 'Reading the virtual machine settings…',
  stateOn: 'On: virtual machines on this PC can use this license.',
  stateOnWaiting: 'On, but the helper is not running yet. It starts the next time you sign in to Windows.',
  stateOff: 'Off: virtual machines on this PC need their own license.',
  noHelper: 'This copy of ScadaBench does not include the helper program, so sharing cannot be turned on.',
  turnOnFailed: 'Could not change this setting',
  firewallTitle: 'Windows Firewall',
  firewallChecking: 'Checking whether your virtual machines can reach this PC…',
  firewallOpen: 'Windows Firewall already lets your virtual machines ask this PC.',
  firewallClosed: 'Your virtual machines cannot reach the helper yet: Windows Firewall stops them.',
  firewallWhy: 'ScadaBench can fix this for you. It adds one Windows Firewall rule that lets only your virtual '
    + 'network reach one port, and Settings › Network › Clean up removes it again.',
  firewallChanges: 'What ScadaBench will change',
  firewallReview: 'Allow through Windows Firewall…',
  firewallAdmin: 'Windows will ask for administrator permission (a UAC prompt). Apply this change?',
  firewallApply: 'Yes, apply',
  firewallCancel: 'Cancel',
  firewallFailed: 'Windows Firewall could not be checked',
  insideVm: 'Licensed through your host PC',
  insideVmHint: 'This virtual machine uses the lifetime license of the PC it runs on. That PC must be switched on '
    + 'and have "Let my virtual machines use this license" turned on in Settings › License.',
  hostKeyRow: 'Host key',
  activationTitle: 'Activation code',
  activationHint: 'Use this code instead of the Machine ID on the offline activation page. It also lets your '
    + 'virtual machines use this license later.',
  copyActivation: 'Copy activation code',
  copiedActivation: 'Copied',
  gateTitle: 'Using a virtual machine?',
  gateIntro: 'A virtual machine (VM) on a licensed PC needs no key, no file and no internet. It uses the license '
    + 'of the PC it runs on (Hyper-V, VMware or VirtualBox).',
  gateSteps: [
    'On the PC that runs this VM: activate ScadaBench there first, if it is not activated yet.',
    'On that PC: open Settings › License and tick "Let my virtual machines use this license". If ScadaBench offers '
      + 'a Windows Firewall rule, apply it.',
    'Come back to this VM. This screen continues on its own within a few seconds. Nothing to type here.',
  ],
  gateLooking: 'Looking for a licensed PC on the virtual network…',
  gateNote: 'The VM keeps working offline as long as that PC is switched on with this option ticked. A VM copied to '
    + 'another computer stops.',
};

/** The status of a virtual machine that is licensed through its host PC. */
export function vmHostState(status: LicenseStatus): Status {
  const key = status.host_key ? ` Host key: ${status.host_key}.` : '';
  return { label: VM_HOST_COPY.insideVm, tone: 'good', hint: VM_HOST_COPY.insideVmHint + key };
}

/** One plain sentence about the switch for the section. */
export function vmHostSummary(enabled: boolean, running: boolean, helperFound: boolean): string {
  if (!helperFound && !enabled) return VM_HOST_COPY.noHelper;
  if (!enabled) return VM_HOST_COPY.stateOff;
  return running ? VM_HOST_COPY.stateOn : VM_HOST_COPY.stateOnWaiting;
}

/** How often the activation screen asks again while a virtual machine may be looking for its host. */
export const VM_HOST_NOTICE_SECONDS = 6;
