// One place for names, prices and addresses used across the site.
export const SITE = {
  name: 'ScadaBench',
  tagline: 'IEC 61850 IED simulator & SCADA test bench',
  url: 'https://scadabench.com',
  licenseServer: 'https://license.scadabench.com',
  download: '/download/ScadaBench-Setup.exe',
  portable: '/download/ScadaBench-Portable.exe',
  supportEmail: 'support@scadabench.com',
  price: '149',
  currency: 'USD',
  trialDays: 7,
  refundDays: 7,
  movesPerYear: 2,
  version: '0.1.6',
  requirements: 'Windows 10 or 11, 64-bit. 4 GB RAM. No Docker, Python or Node needed.',
};

// Main navigation, in reading order. The header shows it on wide screens and in the
// phone menu; Buy and Free trial are separate buttons next to it.
export const NAV = [
  { href: '/', label: 'Home' },
  { href: '/features/', label: 'Features' },
  { href: '/tutorials/', label: 'Tutorials' },
  { href: '/pricing/', label: 'Pricing' },
  { href: '/download/', label: 'Download' },
  { href: '/faq/', label: 'FAQ' },
  { href: '/contact/', label: 'Contact' },
];

export const isCurrent = (href: string, path: string) => (href === '/' ? path === '/' : path.startsWith(href));

export const buyUrl = `${SITE.licenseServer}/buy`;
export const offlineUrl = `${SITE.licenseServer}/activate`;
