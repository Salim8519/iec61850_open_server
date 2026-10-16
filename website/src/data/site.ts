// One place for names, prices and addresses used across the site.
export const SITE = {
  name: 'ScadaBench',
  tagline: 'IEC 61850 IED simulator & SCADA test bench',
  url: 'https://scadabench.com',
  licenseServer: 'https://license.scadabench.com',
  download: '/download/ScadaBench-Setup.exe',
  supportEmail: 'support@scadabench.com',
  price: '149',
  currency: 'USD',
  trialDays: 7,
  movesPerYear: 2,
  version: '0.1.0',
  requirements: 'Windows 10 or 11, 64-bit. 4 GB RAM. No Docker, Python or Node needed.',
};

export const NAV = [
  { href: '/features/', label: 'Features' },
  { href: '/tutorials/', label: 'Tutorials' },
  { href: '/pricing/', label: 'Pricing' },
  { href: '/faq/', label: 'FAQ' },
];

export const buyUrl = `${SITE.licenseServer}/buy`;
export const offlineUrl = `${SITE.licenseServer}/activate`;
