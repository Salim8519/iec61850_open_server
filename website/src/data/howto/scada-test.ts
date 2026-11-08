import type { HowTo } from './types';
import devices from '../../assets/screens/devices.jpg';
import bay from '../../assets/screens/diagram-bay.jpg';

export const SCADA_TEST: HowTo = {
  slug: 'test-scada-without-real-ieds',
  title: 'How to test SCADA without real IEDs',
  short: 'Test SCADA without real IEDs',
  description: 'Test a substation SCADA, HMI or gateway before the relays arrive: simulate every IED from the SCD over IEC 61850, or serve them over IEC 104, DNP3 or Modbus TCP.',
  answer: 'Simulate every IED of the substation from its SCD in ScadaBench and press Run. A SCADA or gateway that speaks IEC 61850 connects to each simulated IED over MMS, exactly as on site. A SCADA that speaks IEC 60870-5-104, DNP3 or Modbus TCP connects to ScadaBench as the station gateway instead. Then you switch breakers, move measurements and send commands, and watch the SCADA react.',
  needs: [
    'The SCD of the substation (or CID files of the IEDs).',
    'ScadaBench on a Windows 10 or 11 PC (free 7-day trial). The SCADA can run on the same PC or on another PC on the same network.',
    'The point list of your SCADA, if you test over IEC 104, DNP3 or Modbus TCP.',
  ],
  steps: [
    { title: 'Simulate every IED',
      text: 'Import the SCD, tick every IED and keep the role Simulate. Each IED becomes an IEC 61850 MMS server with the data model, datasets, report control blocks and controls written in the file.',
      image: { src: devices, alt: 'Devices page listing six running IEDs from an SCD, each with its MMS address' } },
    { title: 'Put the IEDs where your SCADA looks for them',
      text: 'For a SCADA on the same PC, This PC only is enough. For the addresses written in the SCD, use Settings › Put my IEDs on a network › ScadaBench Lab; for a SCADA on another PC, pick a real network card. Every Windows change is shown before it is made.' },
    { title: 'IEC 61850 SCADA: connect and press Run',
      text: 'Press Run and connect the SCADA to each IED on port 102. Reports are sent as the SCD defines them, and controls follow the control model of each object, select before operate included.' },
    { title: 'IEC 104, DNP3 or Modbus SCADA: start the gateway',
      text: 'Open Tools › SCADA gateway & master and press Start gateway. The point list is proposed from your IEDs: positions, alarms and trips, measurements, health and commands. Add points by hand or with Paste from Excel, and import or export the list as CSV.' },
    { title: 'Drive the test',
      text: 'Operate breakers on the single-line diagram, change measurements in Test › Test points, or let generators and timed sequences move them. Commands from the SCADA operate the IEDs over IEC 61850 with their own control model, and the traffic and command logs show every exchange.',
      image: { src: bay, alt: 'Close-up of a running feeder bay on the single-line diagram: disconnectors, breaker and current transformer' } },
    { title: 'Testing a gateway or RTU instead?',
      text: 'The SCADA master in the same tool plays the control centre: an IEC 104 controlling station, a DNP3 master or a Modbus client. It finds every point of your gateway or RTU by itself and marks each one received, missing or wrong type.' },
  ],
  problems: [
    { q: 'The SCADA on another PC cannot connect.',
      a: 'Check that the IEDs or the gateway listen on a network card the SCADA PC can reach, not on This PC only. When the gateway listens on All network adapters, ScadaBench opens Windows Firewall for it after a review.' },
    { q: 'Which SCADA systems work?',
      a: 'Any SCADA or client that speaks IEC 61850 MMS, IEC 60870-5-104, DNP3 or Modbus TCP. ScadaBench is tested between two PCs against third-party IEC 104, DNP3 and Modbus tools and IEC 61850 clients. Tell us which SCADA you use.' },
    { q: 'Can I test on the live substation network?',
      a: 'No. ScadaBench is a laboratory and engineering tool. Never connect it to an operational substation network.' },
  ],
  video: 'scada-gateway',
  updated: '2026-11-08',
};
