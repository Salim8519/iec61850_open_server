// Guided tour wording. Each step points at an element marked data-tour="<target>"
// and may switch to the place where that element lives.
import type { PlaceId } from '../app/places';

export type TourStep = { id: string; title: string; text: string; target?: string; place?: PlaceId };

export const TUTORIALS_URL = 'https://scadabench.com/tutorials/';

export const TOUR_STEPS: TourStep[] = [
  { id: 'welcome', title: 'Welcome to ScadaBench',
    text: 'ScadaBench turns your relay configuration files into live IEC 61850 IEDs on this PC. This tour takes one minute; you can leave it at any time with Esc.' },
  { id: 'devices', title: 'Devices: bring in your IEDs', target: 'devices-import', place: 'devices',
    text: 'Import a CID, ICD or SCD file and tick the IEDs you want. No file yet? Start with a sample IED instead.' },
  { id: 'diagram', title: 'SLD: your single-line diagram', target: 'diagram-build', place: 'diagram',
    text: 'SLD means single-line diagram. Build it from the SCD in one click, or add breakers and switches yourself and link them to IED signals.' },
  { id: 'run', title: 'Run: the IEDs come alive', target: 'run',
    text: 'Press Run. Every IED answers on its own address, port 102, so your SCADA or client can connect. Stop ends the run.' },
  { id: 'test', title: 'Test: change values on purpose', target: 'place-test', place: 'test',
    text: 'Save the alarms and measurements you change most, run generators and timed sequences, and replay CSV traces.' },
  { id: 'settings', title: 'Settings: network and license', target: 'place-settings', place: 'settings',
    text: 'Put IEDs on a real network adapter for other devices, set up GOOSE, and see your license.' },
  { id: 'help', title: 'Help is always here', target: 'help',
    text: 'Reopen this tour or watch the short video tutorials from Help at any time. Happy testing!' },
];

export const TOUR_COPY = {
  next: 'Next', back: 'Back', skip: 'Skip tour', done: 'Start using ScadaBench',
  progress: (step: number, total: number) => `Step ${step} of ${total}`,
  helpTitle: 'Help', takeTour: 'Take the tour', tutorials: 'Video tutorials',
};
