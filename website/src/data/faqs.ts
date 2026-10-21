// Every question the site answers, grouped by topic. The FAQ page shows all groups; the
// pricing page shows the buying group, so both always say the same thing.
import { SITE } from './site';

export interface Faq { q: string; a: string; }
export interface FaqGroup { id: string; title: string; items: Faq[]; }

export const ONE_COMPUTER_POLICY = `Each license works on one computer only. It cannot be moved to another computer. Reinstalling Windows on the same computer is fine. A new computer or a new motherboard needs a new license.`;

export const REFUND_POLICY = `If you have not activated the license yet, we refund you in full through PayPal within ${SITE.refundDays} days of purchase. Email ${SITE.supportEmail} with your license key; the key is deactivated on refund. No refunds after a license is activated. Use the free ${SITE.trialDays}-day trial to test ScadaBench with your own SCD first.`;

export const FAQ_GROUPS: FaqGroup[] = [
  { id: 'start', title: 'Getting started', items: [
    { q: 'Do I need any hardware?', a: 'No. ScadaBench runs the IEDs on your Windows PC. Real relays are only needed if you want to compare them with the SCD.' },
    { q: 'Which Windows versions are supported?', a: SITE.requirements },
    { q: 'Does it work without internet?', a: 'Yes. Internet is only used to start the trial or activate a key. Offline PCs can be activated with a license file.' },
    { q: 'Where do I learn how to use it?', a: 'The tutorials page has nine short narrated videos on the real app, each with the steps in text. The app also opens a guided tour on first start.' },
  ] },
  { id: 'technical', title: 'Technical', items: [
    { q: 'Which files can I import?', a: 'CID, ICD, IID and SCD files in SCL 2003, 2007B and 2007B4. Original SCL and vendor extensions are kept.' },
    { q: 'Can my SCADA or gateway connect to the simulated IEDs?', a: 'Yes. Every IED is a real MMS server on port 102. On "This PC only" mode they listen on private loopback addresses; the lab network setup puts them on a real adapter for other machines.' },
    { q: 'Can ScadaBench operate a real relay?', a: 'Only when you ask it to. IED Explorer sends a command after a short confirmation, and the gateway simulator forwards SCADA commands while "Allow SCADA commands to real IEDs" is on. Reading uses free report blocks or polling, and settings are never written.' },
    { q: 'Does it publish GOOSE and Sampled Values?', a: 'Simulated IEDs publish GOOSE. Sampled Values are listened to and supervised, not published. Both need the free Npcap driver.' },
    { q: 'Is IEC 60870-5-104 supported?', a: 'Yes, in the gateway simulator, together with Modbus TCP and DNP3: your SCADA connects to ScadaBench as the station gateway. It is tested against independent IEC 104 and DNP3 master implementations; tell us which SCADA you use.' },
    { q: 'Can I use it on an operational substation network?', a: 'No. ScadaBench is a laboratory and engineering tool. Never connect it to a live operational network.' },
  ] },
  { id: 'buying', title: 'Licensing and buying', items: [
    { q: 'Can I try it before buying?', a: `Yes. The free ${SITE.trialDays}-day trial has every feature and needs no card. One trial per computer.` },
    { q: 'Is it a subscription?', a: 'No. You pay once and the license does not expire on that PC.' },
    { q: 'Can I get a refund?', a: REFUND_POLICY },
    { q: 'What does the license cover?', a: `One lifetime license covers one computer, with no time limit. ${ONE_COMPUTER_POLICY}` },
    { q: 'Can I move my license to another computer?', a: `No. ${ONE_COMPUTER_POLICY}` },
    { q: 'My substation PC has no internet. Can I still activate?', a: `Yes. Copy the Machine ID from ScadaBench, activate on any other computer at ${SITE.licenseServer.replace('https://', '')}/activate and load the license file on the offline PC.` },
    { q: 'Can I use ScadaBench in a virtual machine?', a: 'Yes, on your licensed PC. Tick "Let my virtual machines use this license" in Settings › License and every VM on that PC (Hyper-V, VMware, VirtualBox) uses the same license, with no second key and no internet. A VM copied to another computer stops. See the picture guide on the tutorials page.' },
    { q: 'Can my company buy several licenses or get an invoice?', a: `Yes. Write to ${SITE.supportEmail} with the number of licenses and your company details.` },
  ] },
];

export const faqSchema = (items: Faq[]) => ({
  '@context': 'https://schema.org', '@type': 'FAQPage',
  mainEntity: items.map((item) => ({ '@type': 'Question', name: item.q, acceptedAnswer: { '@type': 'Answer', text: item.a } })),
});
