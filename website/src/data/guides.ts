// Picture guides: short step-by-step help with screenshots of the real app (demo data only), for
// topics a video does not fit. Images live in public/guides; size is the image's pixel size.
export interface GuideStep { text: string; image?: { src: string; alt: string; width: number; height: number }; }
export interface Guide { slug: string; title: string; intro: string; steps: GuideStep[]; note: string; }

export const GUIDES: Guide[] = [
  { slug: 'virtual-machines', title: 'Use ScadaBench in a virtual machine',
    intro: 'A virtual machine (VM) on your licensed PC needs no key, no file and no internet: it uses the license of the PC it runs on. Works with Hyper-V, VMware and VirtualBox.',
    steps: [
      { text: 'On your PC (the host), activate ScadaBench as usual. Then open Settings › License and tick "Let my virtual machines use this license". If ScadaBench offers a Windows Firewall rule, apply it.',
        image: { src: '/guides/vm-0-host-settings.webp', alt: 'Settings › License on the host PC with "Let my virtual machines use this license" ticked and switched on', width: 1400, height: 657 } },
      { text: 'In the VM, install ScadaBench and open it. The activation screen shows two options: "Server & offline options" and "Using a virtual machine?". You do not need to type anything.',
        image: { src: '/guides/vm-1-activation-closed.webp', alt: 'The activation screen inside the VM with the offline and virtual machine options closed', width: 1400, height: 977 } },
      { text: 'Open "Using a virtual machine?" to see the same steps. ScadaBench keeps looking for your licensed PC by itself, even while this section is closed.',
        image: { src: '/guides/vm-2-activation-vm-open.webp', alt: 'The "Using a virtual machine?" section opened, with three steps and a "Looking for a licensed PC" line', width: 1400, height: 1322 } },
      { text: 'Within a few seconds the VM opens ScadaBench. Settings › License in the VM shows "Licensed through your host PC" and the same host key as on your PC.',
        image: { src: '/guides/vm-3-vm-licensed.webp', alt: 'Settings › License inside the VM showing "Licensed through your host PC" and the host key', width: 1400, height: 390 } },
    ],
    note: 'The VM keeps working offline while your PC is switched on with the option ticked. A VM copied to another computer stops. Untick the option any time to stop sharing.' },
];
