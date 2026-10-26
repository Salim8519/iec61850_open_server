// Tutorials: a short captioned video of the real app (rendered by marketing/videos) plus the steps in text.
// seconds is the measured length of the rendered video (ffprobe on marketing/videos/out/web).
export interface Tutorial { slug: string; title: string; seconds: number; steps: string[]; }

export const videoOf = (tutorial: Tutorial) => `/videos/${tutorial.slug}.mp4`;
export const posterOf = (tutorial: Tutorial) => `/videos/${tutorial.slug}.jpg`;
export const lengthOf = (seconds: number) => `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;

export const TUTORIALS: Tutorial[] = [
  { slug: 'getting-started', title: 'IEC 61850 and ScadaBench in one minute', seconds: 111, steps: [
    'IEC 61850 is the standard substation devices (IEDs) use to talk. SCADA reads them over MMS, TCP port 102.',
    'Every IED has a data model: logical devices, logical nodes such as XCBR (breaker), and data such as Pos.stVal.',
    'Devices › Use a sample IED, then Run: the IED answers on its own address, like the real relay.',
  ] },
  { slug: 'cid-scd', title: 'CID and SCD files', seconds: 82, steps: [
    'SCL files describe IEC 61850 projects. A CID describes one configured IED; an SCD describes the whole substation.',
    'Devices › Import CID / SCD, tick the IEDs you want. Each IED gets a role: Simulate, Monitor real device or Skip.',
    'The list shows who sends GOOSE to whom. File check details lists anything unusual; nothing is dropped.',
  ] },
  { slug: 'sld', title: 'The single-line diagram (SLD)', seconds: 85, steps: [
    'An SLD draws a three-phase station with one line per circuit: busbars, bays, breakers (QA), disconnectors (QB), earthing switches (QC).',
    'SLD › Build from SCD › Preview, then Apply. Tidy up arranges it; one Undo restores your layout.',
    'Press Run, click a breaker: Operate sends a real IEC 61850 command (select before operate in one press); Test moves it by itself, with no command.',
  ] },
  { slug: 'simulation', title: 'Simulate IEDs and change values', seconds: 54, steps: [
    'Simulation means ScadaBench plays the IED; your SCADA or gateway connects to it like to the real relay.',
    'Test › Test points: type chips show each kind of point; add the breaker position and measurements you change most.',
    'Press Run and change them live. Generators move values for you; Sequences play timed steps or a CSV trace.',
  ] },
  { slug: 'goose', title: 'GOOSE and the lab network', seconds: 79, steps: [
    'GOOSE is the fast multicast message IEDs send each other for trips and interlocks: one publisher, many subscribers.',
    'Each IED lists its GOOSE partners; Signal flow draws who sends GOOSE and reports to whom.',
    'Settings › Put my IEDs on a network: pick the network card, review every Windows change, then apply. Tools shows the GOOSE really on the wire.',
  ] },
  { slug: 'real-devices', title: 'Connect to real devices', seconds: 59, steps: [
    'Simulate is for when the relay is not there; Monitor real device is for when it is. ScadaBench connects as a client and only reads.',
    'Enter the device address and press Run. It uses a report block only if it is free, otherwise it polls, and gives it back on Stop.',
    'Compare with SCD lists every difference; Health check gives one verdict per IED, exportable as CSV or HTML.',
  ] },
  { slug: 'ied-explorer', title: 'IED Explorer and Find IEDs', seconds: 80, steps: [
    'Tools › Find IEDs scans a network for port 102 and asks each device for its vendor, model and logical devices.',
    'Open a result in IED Explorer: it browses the data model and watches key values, without taking a report block.',
    'Commands are sent only after one confirmation, with the device’s own control model: Select, then Operate within the time limit. Export the watch list as CSV.',
  ] },
  { slug: 'scada-gateway', title: 'Be the SCADA gateway', seconds: 91, steps: [
    'Tools › SCADA gateway & master: ScadaBench serves your IEDs to SCADA over IEC 104, Modbus TCP or DNP3.',
    'Add points from an IED, made-up points or rows pasted from Excel, or press Start gateway: the point list is proposed from your IEDs.',
    'SCADA commands, or the Send buttons, operate the IEDs over real IEC 61850. SCADA master checks your point list: received, missing or wrong type.',
  ] },
  { slug: 'goose-sld', title: 'Send GOOSE from the SLD', seconds: 89, steps: [
    'SLD › Add › Add all GOOSE IEDs: one box per IED that sends or receives GOOSE, in one Undo step.',
    'Press Run and switch on the GOOSE messages layer. Click an IED box: its sheet lists each message, value by value, and who receives it.',
    'Change a value to send it: the line runs from sender to receivers, and “got it” shows the new message arrived. A breaker or point panel row can send GOOSE too.',
  ] },
  { slug: 'point-panels', title: 'Point panels and the smart point picker', seconds: 102, steps: [
    'SLD › Add › Point panel: one card for the points of one IED. Add points by type (status, double point, value, command) and by object, such as XCBR1.',
    'Press Run and open the panel: switch statuses, set values and send commands. All healthy puts every point back in one go.',
    'Every SLD symbol uses the same picker: it preselects the type the symbol needs, says why, and counts each type.',
  ] },
  { slug: 'commands', title: 'Select before operate, typed commands and tap changers', seconds: 98, steps: [
    'ScadaBench reads each control model from the SCD. Command options › Show Select before Operate step by step: Select, a countdown, then Operate or Cancel.',
    'Add a tap-changer transformer, link its tap position and its raise/lower step command (BSC or ISC), and tick the simulated voltage regulator.',
    'Press Run: Raise and Lower send real step commands; in Auto the regulator keeps the LV voltage in its band.',
  ] },
  { slug: 'snmp', title: 'SNMP devices: switches and clocks', seconds: 51, steps: [
    'Tools › SNMP devices › Simulate devices: add a network switch or a GPS time server and press Start devices.',
    'Each device answers SNMP v1, v2c and v3 on its own address, port 161, with named values that change like the real device.',
    'Read devices reads any SNMP device: one full walk, the important values on top, re-read by itself, and traps received.',
  ] },
  { slug: 'activation', title: 'Activate or go offline', seconds: 56, steps: [
    'A new install opens on the activation screen: start the free 7-day trial, or paste the key from your purchase email.',
    'Each license works on one computer only and cannot be moved. Reinstalling Windows on the same computer is fine; a new computer needs its own license.',
    'No internet on site? Copy the Machine ID, get a license file at license.scadabench.com/activate and load it.',
  ] },
  { slug: 'lab-network', title: 'IEDs at their own addresses, without a VM', seconds: 64, steps: [
    'Settings › Lab network on this PC: Windows asks once to add the ScadaBench Lab card (no download, no restart). Then press Use the lab network; no more prompts.',
    'Each simulated IED answers at its own address from the SCD. Type another address and press Enter to move it, even while running; the SCD file is never changed.',
    'Ping, IEDScout, your SCADA and GOOSE reach the IEDs on this PC. Addresses used by this PC or by a real device on your Ethernet or Wi-Fi are caught before they clash.',
  ] },
];

export const totalSeconds = TUTORIALS.reduce((sum, tutorial) => sum + tutorial.seconds, 0);
