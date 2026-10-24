// One place for names, prices and addresses used across the site.
const VERSION = '0.1.27';
export const SITE = {
  name: 'ScadaBench',
  tagline: 'IEC 61850 IED simulator & SCADA test bench',
  url: 'https://scadabench.com',
  licenseServer: 'https://license.scadabench.com',
  // Files carry the version (deploy.cmd publishes them under these names; old names redirect).
  // The official installer, and the portable exe that runs directly with no install (ADR-0063).
  // The portable exe is over the 25 MiB Pages file limit, so deploy.cmd puts it in the
  // scadabench-downloads R2 bucket served at download.scadabench.com.
  download: `/download/ScadaBench-Setup-${VERSION}.exe`,
  portable: `https://download.scadabench.com/download/ScadaBench-Portable-${VERSION}.exe`,
  supportEmail: 'support@scadabench.com',
  price: '149',
  currency: 'USD',
  trialDays: 7,
  refundDays: 7,
  version: VERSION,
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
