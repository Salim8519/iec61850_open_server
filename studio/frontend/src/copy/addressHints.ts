// Words for a typed address the page checks at once, before the engine answers. They match the engine's own
// sentences, so the line does not change when the engine's answer arrives.
export const duplicateHint = (owner: string, ip: string, noun: 'IED' | 'device') =>
  `${owner} already uses ${ip}. Each ${noun} needs its own address.`;
export const otherCardHint = (ip: string, card: string) =>
  `${ip} already belongs to '${card}'. Pick another address or disconnect that adapter.`;
export const CHECKING_HINT = 'Checking this address…';
