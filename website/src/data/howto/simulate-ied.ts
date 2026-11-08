import type { HowTo } from './types';
import devices from '../../assets/screens/devices.jpg';
import bay from '../../assets/screens/diagram-bay.jpg';

export const SIMULATE_IED: HowTo = {
  slug: 'simulate-iec-61850-ied-from-cid-scd',
  title: 'How to simulate an IEC 61850 IED from a CID or SCD file',
  short: 'Simulate an IED from a CID or SCD file',
  description: 'Turn a CID, ICD, IID or SCD file into a live IEC 61850 MMS server on Windows, with reports and GOOSE, so SCADA and clients can connect without the real relay.',
  answer: 'Import the file into ScadaBench, give the IED the role Simulate and press Run. The IED becomes a live MMS server on TCP port 102, built from the exact data model in your file, with its datasets, reports, controls and GOOSE. Your SCADA, gateway or IEC 61850 client connects to it as it would to the real relay.',
  needs: [
    'A Windows 10 or 11 PC (64-bit). No relay, Docker, Python or Node.',
    'Your CID, ICD, IID or SCD file (SCL 2003, 2007B or 2007B4). No file yet? Devices › Use a sample IED gives you one.',
    'ScadaBench with the free 7-day trial. Every feature works in the trial.',
  ],
  steps: [
    { title: 'Import the file',
      text: 'Open Devices › Import CID / SCD and pick your file. An SCD lists every IED in the substation; tick the ones you want. Original SCL and vendor extensions are kept, and File check details lists anything unusual instead of dropping it.' },
    { title: 'Give each IED a role',
      text: 'Choose Simulate for each IED that ScadaBench should play. Monitor real device is for a relay that is on the bench, and Skip keeps an IED in the project without starting it. The list also shows who sends GOOSE to whom.',
      image: { src: devices, alt: 'Devices page listing six running IEDs from an SCD, each with its MMS address' } },
    { title: 'Choose where the IEDs listen',
      text: 'On This PC only, each IED listens on a private address of this PC, which is enough for a SCADA or client on the same computer. To reach each IED at the address written in the SCD, without a VM, use Settings › Put my IEDs on a network › ScadaBench Lab. To let other machines connect, pick a real network card instead; every Windows change is shown before it is made.' },
    { title: 'Press Run and connect',
      text: 'Press Run. Each IED answers IEC 61850 MMS on port 102, sends its reports and polling answers, and publishes its GOOSE. Point your SCADA, gateway or client (IEDScout, for example) at the IED address and browse the model as you would on the real device.',
      image: { src: bay, alt: 'Close-up of a running feeder bay on the single-line diagram: disconnectors, breaker and current transformer' } },
    { title: 'Change values while it runs',
      text: 'Test › Test points holds the breaker positions and measurements you change most. Generators move values for you, and Sequences play timed steps or a CSV trace. On the single-line diagram, Operate on a breaker sends a real IEC 61850 command; Test moves it by itself, with no command.' },
  ],
  problems: [
    { q: 'Run says port 102 is in use.',
      a: 'Another program already listens on port 102 at that address. ScadaBench names the program that holds the port and what Windows answered. Stop that program, or move the IED to another address.' },
    { q: 'My SCADA on another PC cannot reach the IED.',
      a: 'This PC only addresses are private to this computer. Put the IEDs on a real network card under Settings › Put my IEDs on a network, and check that the SCADA PC is on the same network.' },
    { q: 'The file check lists warnings.',
      a: 'They describe what is unusual in the file, such as a subscription that points nowhere. Nothing is removed; the IED runs with the model exactly as written.' },
  ],
  video: 'cid-scd',
  updated: '2026-11-08',
};
