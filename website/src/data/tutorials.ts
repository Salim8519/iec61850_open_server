// Tutorials: a short captioned video of the real app (rendered by marketing/videos) plus the steps in text.
// seconds is the measured length of the rendered video (ffprobe on marketing/videos/out/web).
export interface Tutorial { slug: string; title: string; seconds: number; steps: string[]; }

export const videoOf = (tutorial: Tutorial) => `/videos/${tutorial.slug}.mp4`;
export const posterOf = (tutorial: Tutorial) => `/videos/${tutorial.slug}.jpg`;
export const lengthOf = (seconds: number) => `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;

export const TUTORIALS: Tutorial[] = [
  { slug: 'getting-started', title: 'IEC 61850 and ScadaBench in one minute', seconds: 95, steps: [
    'IEC 61850 is the standard substation devices (IEDs) use to talk. SCADA reads them over MMS, TCP port 102.',
    'Every IED has a data model: logical devices, logical nodes such as XCBR (breaker), and data such as Pos.stVal.',
    'Devices › Use a sample IED, then Run: the IED answers on its own address, like the real relay.',
  ] },
  { slug: 'cid-scd', title: 'CID and SCD files', seconds: 73, steps: [
    'SCL files describe IEC 61850 projects. A CID describes one configured IED; an SCD describes the whole substation.',
    'Devices › Import CID / SCD, tick the IEDs you want. Each IED gets a role: Be this IED, Read real IED or Leave out.',
    'The list shows who sends GOOSE to whom. File check details lists anything unusual; nothing is dropped.',
  ] },
  { slug: 'sld', title: 'The single-line diagram (SLD)', seconds: 70, steps: [
    'An SLD draws a three-phase station with one line per circuit: busbars, bays, breakers (QA), disconnectors (QB), earthing switches (QC).',
    'SLD › Build from SCD › Preview, then Apply. Tidy up arranges it; one Undo restores your layout.',
    'Press Run, click a breaker: Operate sends a real IEC 61850 command; Test moves it by itself, with no command.',
  ] },
  { slug: 'simulation', title: 'Simulate IEDs and change values', seconds: 45, steps: [
    'Simulation means ScadaBench plays the IED; your SCADA or gateway connects to it like to the real relay.',
    'Test › Test points: add the breaker position and measurements you change most.',
    'Press Run and change them live. Generators move values for you; Sequences play timed steps or a CSV trace.',
  ] },
  { slug: 'goose', title: 'GOOSE and the lab network', seconds: 67, steps: [
    'GOOSE is the fast multicast message IEDs send each other for trips and interlocks: one publisher, many subscribers.',
    'Each IED lists its GOOSE partners; Signal flow draws who sends GOOSE and reports to whom.',
    'Settings › Put my IEDs on a network: pick the network card, review every Windows change, then apply. Tools shows the GOOSE really on the wire.',
  ] },
  { slug: 'real-devices', title: 'Connect to real devices', seconds: 57, steps: [
    'Be this IED is for when the relay is not there; Read real IED is for when it is. ScadaBench connects as a client and only reads.',
    'Enter the device address and press Run. It uses a report block only if it is free, otherwise it polls, and gives it back on Stop.',
    'Compare with SCD lists every difference; Health check gives one verdict per IED, exportable as CSV or HTML.',
  ] },
  { slug: 'ied-explorer', title: 'IED Explorer and Find IEDs', seconds: 72, steps: [
    'Tools › Find IEDs scans a network for port 102 and asks each device for its vendor, model and logical devices.',
    'Open a result in IED Explorer: it browses the data model and watches key values, without taking a report block.',
    'Commands are sent only after one confirmation, with the device’s own control model. Export the watch list as CSV.',
  ] },
  { slug: 'scada-gateway', title: 'Be the SCADA gateway', seconds: 75, steps: [
    'Tools › SCADA gateway & master: ScadaBench serves your IEDs to SCADA over IEC 104, Modbus TCP or DNP3.',
    'Press Start gateway: the point list is proposed from your IEDs, with addresses and live values.',
    'SCADA commands, or the Send buttons, operate the IEDs over real IEC 61850. SCADA master and Protocol converter test the other side.',
  ] },
  { slug: 'activation', title: 'Activate or go offline', seconds: 46, steps: [
    'A new install opens on the activation screen: start the free 7-day trial, or paste the key from your purchase email.',
    'Each license works on one computer only and cannot be moved. Reinstalling Windows on the same computer is fine; a new computer needs its own license.',
    'No internet on site? Copy the Machine ID, get a license file at license.scadabench.com/activate and load it.',
  ] },
];

export const totalSeconds = TUTORIALS.reduce((sum, tutorial) => sum + tutorial.seconds, 0);
