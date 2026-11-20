// The manual's sections, in reading order, and the helpers that order and link its pages.
// Pages are Markdown files in src/content/docs/<section>/<page>.md (see content.config.ts).
export const DOC_SECTIONS = [
  { id: 'getting-started', title: 'Getting started', blurb: 'Install ScadaBench, start it and run your first IED.' },
  { id: 'devices', title: 'Devices', blurb: 'Import CID and SCD files, simulate IEDs and connect to real ones.' },
  { id: 'sld', title: 'SLD', blurb: 'Draw the single-line diagram and link it to IED signals.' },
  { id: 'test', title: 'Test', blurb: 'Change values, run generators, sequences and CSV playback.' },
  { id: 'tools', title: 'Tools', blurb: 'Find IEDs, the SCADA gateway, SNMP devices and network traffic.' },
  { id: 'network', title: 'Network', blurb: 'Put simulated IEDs on a network adapter with their own addresses.' },
  { id: 'licence', title: 'Licence and VMs', blurb: 'Trial, activation, offline licences and virtual machines.' },
  { id: 'troubleshooting', title: 'Troubleshooting', blurb: 'Fix the most common problems, step by step.' },
] as const;

export const DOC_SECTION_IDS = DOC_SECTIONS.map((section) => section.id) as [string, ...string[]];

export type DocSectionId = (typeof DOC_SECTIONS)[number]['id'];

export interface DocEntry {
  id: string;
  data: { title: string; description: string; section: string; order: number; video?: string };
}

export const sectionOf = (id: string) => DOC_SECTIONS.find((section) => section.id === id);

/** Pages in manual order: by section (as listed above), then by the order number in each page. */
export function sortDocs<T extends DocEntry>(entries: T[]): T[] {
  const rank = (entry: T) => DOC_SECTIONS.findIndex((section) => section.id === entry.data.section);
  return [...entries].sort((a, b) => rank(a) - rank(b) || a.data.order - b.data.order);
}

/** The page name in the URL is the file name without .md (the folder is the section). */
export const slugOf = (entry: DocEntry) => entry.id.split('/').slice(1).join('/');

export const docPath = (entry: DocEntry) => `/docs/${entry.data.section}/${slugOf(entry)}/`;

/** The section heading for the sidebar and index: only sections that have pages, in order. */
export function groupBySection<T extends DocEntry>(sorted: T[]) {
  return DOC_SECTIONS.map((section) => ({ ...section, pages: sorted.filter((entry) => entry.data.section === section.id) }));
}
