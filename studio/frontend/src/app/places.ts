// The five places of the app (ADR-0032, ADR-0048, ADR-0054). Order is the sidebar order: IEDs come
// first because the single-line diagram (SLD) links its symbols to their signals.
export type PlaceId = 'diagram' | 'devices' | 'test' | 'tools' | 'settings';

export const PLACES: { id: PlaceId; title: string; heading: string; subtitle: string }[] = [
  { id: 'devices', title: 'Devices', heading: 'Devices', subtitle: 'Your IEDs: import, simulate, or connect to the real device.' },
  { id: 'diagram', title: 'SLD', heading: 'Single-line diagram (SLD)',
    subtitle: 'Draw your bay and connect each symbol to an IED signal.' },
  { id: 'test', title: 'Test', heading: 'Test', subtitle: 'Change values, run generators and timed sequences.' },
  { id: 'tools', title: 'Tools', heading: 'Tools', subtitle: 'Find IEDs, simulate a gateway, watch GOOSE and SV traffic.' },
  { id: 'settings', title: 'Settings', heading: 'Settings', subtitle: 'Network adapter, license and information about this app.' },
];

export const place = (id: PlaceId) => PLACES.find(item => item.id === id)!;
