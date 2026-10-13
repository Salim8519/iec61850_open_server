// Plain explanations for every technical term the guided screens show.
export type GlossaryEntry = { term: string; plain: string };

export const GLOSSARY: Record<string, GlossaryEntry> = {
  ied: { term: 'IED', plain: 'An intelligent electronic device: the protection relay or controller in a bay.' },
  scd: { term: 'SCD file', plain: 'The substation configuration file that describes every IED, its data and its network addresses.' },
  mms: { term: 'MMS', plain: 'The client/server protocol SCADA, gateways and test tools use to read and control an IED over TCP port 102.' },
  goose: { term: 'GOOSE', plain: 'Fast messages IEDs broadcast to each other on the network, for example trips and interlocking signals.' },
  gocb: { term: 'GOOSE block', plain: 'The part of an IED that decides which values it sends by GOOSE and how often.' },
  appid: { term: 'APPID', plain: 'A number that labels one GOOSE stream so receivers can pick out the messages meant for them.' },
  mac: { term: 'MAC address', plain: 'The hardware destination address of a GOOSE message; GOOSE uses special multicast addresses.' },
  vlan: { term: 'VLAN', plain: 'A tag that keeps different kinds of traffic apart on the same network cable.' },
  confrev: { term: 'Configuration revision', plain: 'A counter that changes when a GOOSE or report configuration changes; both sides must agree.' },
  stnum: { term: 'State number', plain: 'A GOOSE counter that goes up every time a value in the message changes.' },
  tal: { term: 'Time allowed to live', plain: 'How long a receiver waits for the next GOOSE message before it treats the sender as lost.' },
  lgos: { term: 'LGOS', plain: 'A supervision node inside an IED that shows whether a subscribed GOOSE message is arriving.' },
  rcb: { term: 'Report block', plain: 'A slot in a real IED that pushes value changes to one client. Each IED has only a few.' },
  gi: { term: 'General interrogation', plain: 'A request that makes the IED send all current values once, so the screen starts complete.' },
  polling: { term: 'Polling', plain: 'Studio asks the IED for each value again and again. It never changes anything in the IED.' },
  npcap: { term: 'Npcap', plain: 'A free Windows driver that lets programs send and receive GOOSE. You install it once from npcap.com.' },
  adapter: { term: 'Network adapter', plain: 'A network port of this PC, such as a USB Ethernet dongle, the built-in Ethernet port or Wi-Fi.' },
  alias: { term: 'Extra address', plain: 'An additional IP address Windows gives to an adapter so a simulated IED can be reached on it.' },
  firewall: { term: 'Firewall rule', plain: 'A Windows permission that lets other devices connect to the simulated IEDs.' },
  uac: { term: 'Administrator prompt', plain: 'The Windows window that asks for permission before a program changes network settings.' },
  loopback: { term: 'This PC only', plain: 'Private 127.x addresses that only programs on this PC can reach. Safe and needs no setup.' },
  subnet: { term: 'Subnet', plain: 'The range of addresses that devices on the same network share, such as 192.168.1.x.' },
  quality: { term: 'Quality', plain: 'Flags an IED sends with each value to say whether the value can be trusted.' },
};

export const glossaryTerm = (key: string): GlossaryEntry => GLOSSARY[key];
