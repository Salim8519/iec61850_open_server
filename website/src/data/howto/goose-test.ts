import type { HowTo } from './types';
import flow from '../../assets/screens/signal-flow.jpg';
import overview from '../../assets/screens/diagram-overview.jpg';

export const GOOSE_TEST: HowTo = {
  slug: 'test-iec-61850-goose-without-test-set',
  title: 'How to test IEC 61850 GOOSE without a test set',
  short: 'Test GOOSE without a test set',
  description: 'Simulate the IEDs that publish GOOSE from your SCD, send a GOOSE change by hand and check what is really on the wire, on a Windows PC with no test set.',
  answer: 'Simulate the publishing IEDs from your SCD in ScadaBench. Each one publishes the GOOSE messages declared in the file. Change a value and its GOOSE goes out to the subscribers. Share it with a network card and a real relay on the bench receives it, while Tools › GOOSE & Sampled Values traffic shows which GOOSE is really on the wire and checks it against the SCD.',
  needs: [
    'The SCD of the substation, with its GOOSE control blocks and subscriptions.',
    'ScadaBench on a Windows 10 or 11 PC (free 7-day trial).',
    'To send or hear GOOSE on a real network card: the free Npcap driver from npcap.com, installed once.',
  ],
  steps: [
    { title: 'Import the SCD and see who talks to whom',
      text: 'Import the SCD and simulate the IEDs that publish the GOOSE you want to test. Each IED lists its GOOSE partners, and Signal flow lists every GOOSE link declared in the SCD and points out subscriptions that point nowhere.',
      image: { src: flow, alt: 'Signal flow table listing GOOSE senders, datasets and declared receivers' } },
    { title: 'Put the GOOSE IEDs on the drawing',
      text: 'On the single-line diagram, Add › Add all GOOSE IEDs places one box for every IED that sends or receives GOOSE, in one Undo step.',
      image: { src: overview, alt: 'Single-line diagram with busbars and bays built from the SCD Substation section' } },
    { title: 'Press Run and open the GOOSE messages layer',
      text: 'Press Run and switch on the GOOSE messages layer. Click an IED box: its sheet lists each GOOSE message value by value, and who receives it.' },
    { title: 'Send a change',
      text: 'Change a value to send it. The line runs from the sender to its receivers, and “got it” shows that the new message arrived. A breaker or a point panel row can send GOOSE too, so a trip or an interlock signal is one click.' },
    { title: 'Share GOOSE with the network',
      text: 'GOOSE stays inside ScadaBench until you share it. In Settings › Put my IEDs on a network, under Where GOOSE goes, pick Also share with the network and review the warning; the adapter is picked automatically or chosen under Advanced. A smart guard listens for 3 seconds first, never resends network GOOSE, and stops an IED if a real device sends the same GOOSE. Use a test network only.' },
    { title: 'Check what is really on the wire',
      text: 'Tools › GOOSE & Sampled Values traffic shows the GOOSE and Sampled Values messages that are really on the network. Every stream is checked against the SCD (confRev, dataset, goID, needs-commissioning and time allowed to live), with a plain-language fix for each problem.' },
  ],
  problems: [
    { q: 'Nothing is sent to the network.',
      a: 'GOOSE stays inside ScadaBench until you pick Also share with the network under Where GOOSE goes, and a real network card needs the Npcap driver. While it is shared, the top bar says GOOSE is shared with the network.' },
    { q: 'The relay does not accept the GOOSE.',
      a: 'Open Tools › GOOSE & Sampled Values traffic. A different goID, dataset or confRev than the relay expects is listed there with the fix; the SCD of the relay and the publisher must agree.' },
    { q: 'Can ScadaBench publish Sampled Values too?',
      a: 'No. ScadaBench listens to and checks Sampled Values; it does not publish them.' },
  ],
  video: 'goose-sld',
  updated: '2026-11-08',
};
