// Tutorials: a short captioned video of the real app (rendered by marketing/videos) plus the steps in text.
export interface Tutorial { slug: string; title: string; minutes: number; steps: string[]; }

export const videoOf = (tutorial: Tutorial) => `/videos/${tutorial.slug}.mp4`;
export const posterOf = (tutorial: Tutorial) => `/videos/${tutorial.slug}.jpg`;

export const TUTORIALS: Tutorial[] = [
  { slug: 'getting-started', title: 'Getting started in one minute', minutes: 1, steps: [
    'Install ScadaBench and start the free trial.',
    'Follow the short tour that opens on first start; it shows the four places: Diagram, Devices, Test and Settings.',
    'Devices › Add a sample IED, then press Run. The IED answers on 127.50.0.2, port 102.',
  ] },
  { slug: 'import-scd', title: 'Import an SCD and build the diagram', minutes: 2, steps: [
    'Devices › Import CID / SCD and choose your file. Tick the IEDs you want and press Import.',
    'Diagram › Build from SCD › Preview, then Apply to diagram. Busbars and bays come from the Substation section.',
    'Press Tidy up for a clean single-line diagram. One Undo puts the old layout back.',
  ] },
  { slug: 'run-and-test', title: 'Run IEDs and change values', minutes: 2, steps: [
    'Press Run. Every IED gets its own address; copy it from the IED Overview for your SCADA or client.',
    'Test › Test points: add the alarms, statuses and measurements you change often, then change them live.',
    'Test › Sequences: set and check values on a timeline, or play back a CSV trace.',
  ] },
  { slug: 'goose-lab-network', title: 'GOOSE and the lab network', minutes: 2, steps: [
    'Install the free Npcap driver from npcap.com with its default options.',
    'Settings › Network › Set up lab network: pick an adapter, review every Windows change, then Apply.',
    'Press Run. Simulated IEDs publish their GOOSE; Network & GOOSE shows every stream and any mismatch with the SCD.',
  ] },
  { slug: 'real-ieds-health', title: 'Real IEDs and station health', minutes: 2, steps: [
    'On an IED choose Real device and enter its address. ScadaBench connects read-only.',
    'It reads the model, compares it with the SCD and uses only free report blocks.',
    'Devices › Health check gives one verdict per IED; export it as CSV or an HTML report.',
  ] },
  { slug: 'activation', title: 'Activate, move or go offline', minutes: 1, steps: [
    'Paste your key on the activation screen and press Activate this computer.',
    'New PC? Enter the key there and choose Move the key here (twice a year, self-service).',
    'No internet? Copy the Machine ID, get a license file at license.scadabench.com/activate and load it.',
  ] },
];
