// The four places of the app (ADR-0032). Order is the sidebar order.
export type PlaceId = 'diagram' | 'devices' | 'test' | 'settings';

export const PLACES: { id: PlaceId; title: string; subtitle: string }[] = [
  { id: 'diagram', title: 'Diagram', subtitle: 'Draw your bay and connect each symbol to an IED signal.' },
  { id: 'devices', title: 'Devices', subtitle: 'Your IEDs: import, simulate, or connect to the real device.' },
  { id: 'test', title: 'Test', subtitle: 'Change values, run generators and timed sequences.' },
  { id: 'settings', title: 'Settings', subtitle: 'Network, GOOSE and information about this app.' },
];

export const place = (id: PlaceId) => PLACES.find(item => item.id === id)!;
