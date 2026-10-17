// Tutorials: a short captioned video of the real app (rendered by marketing/videos) plus the steps in text.
// seconds is the measured length of the rendered video (ffprobe on marketing/videos/out/web).
export interface Tutorial { slug: string; title: string; seconds: number; steps: string[]; }

export const videoOf = (tutorial: Tutorial) => `/videos/${tutorial.slug}.mp4`;
export const posterOf = (tutorial: Tutorial) => `/videos/${tutorial.slug}.jpg`;
export const lengthOf = (seconds: number) => `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;

export const TUTORIALS: Tutorial[] = [
  { slug: 'getting-started', title: 'IEC 61850 and ScadaBench in one minute', seconds: 86, steps: [
    'IEC 61850 is the standard substation devices (IEDs) use to talk. SCADA reads them over MMS, TCP port 102.',
    'Every IED has a data model: logical devices, logical nodes such as XCBR (breaker), and data such as Pos.stVal.',
    'Devices › Use a sample IED, then Run: the IED answers on its own address, like the real relay.',
  ] },
  { slug: 'cid-scd', title: 'CID and SCD files', seconds: 48, steps: [
    'SCL files describe IEC 61850 projects. A CID describes one configured IED; an SCD describes the whole substation.',
    'The SCD holds every IED, its network addresses, GOOSE and the single-line layout (Substation section).',
    'Devices › Import CID / SCD, tick the IEDs you want. File check details lists anything unusual; nothing is dropped.',
  ] },
  { slug: 'sld', title: 'The single-line diagram (SLD)', seconds: 51, steps: [
    'An SLD draws a three-phase station with one line per circuit: busbars, bays, breakers (QA), disconnectors (QB), earthing switches (QC).',
    'SLD › Build from SCD › Preview, then Apply. Tidy up arranges it; one Undo restores your layout.',
    'Each symbol is linked to its IED signal. Press Run and every switch shows its live position.',
  ] },
  { slug: 'simulation', title: 'Simulate IEDs and change values', seconds: 44, steps: [
    'Simulation means ScadaBench plays the IED; your SCADA or gateway connects to it like to the real relay.',
    'Test › Test points: add the breaker position and measurements you change most (before Run).',
    'Press Run and change them live. Generators move values for you; Sequences play timed steps or a CSV trace.',
  ] },
  { slug: 'goose', title: 'GOOSE and the lab network', seconds: 58, steps: [
    'GOOSE is the fast multicast message IEDs send each other for trips and interlocks: one publisher, many subscribers.',
    'Settings › Network & GOOSE lists every GOOSE block with its APPID and multicast MAC. Install the free Npcap driver.',
    'Set up lab network: pick the adapter, keep the SCD addresses, review every Windows change, then Apply. Run publishes GOOSE.',
  ] },
  { slug: 'real-devices', title: 'Connect to real devices', seconds: 47, steps: [
    'Simulate is for when the relay is not there; Real device is for when it is. ScadaBench connects as a client and only reads.',
    'Enter the device address and press Run. It uses a report block only if it is free, otherwise it polls, and gives it back on Stop.',
    'Compare with SCD lists every difference; Health check gives one verdict per IED, exportable as CSV or HTML.',
  ] },
  { slug: 'activation', title: 'Activate, move or go offline', seconds: 45, steps: [
    'A new install opens on the activation screen: start the free 7-day trial, or paste the key from your purchase email.',
    'New PC? Enter the key there and choose Move the key here (twice a year, self-service).',
    'No internet on site? Copy the Machine ID, get a license file at license.scadabench.com/activate and load it.',
  ] },
];

export const totalSeconds = TUTORIALS.reduce((sum, tutorial) => sum + tutorial.seconds, 0);
