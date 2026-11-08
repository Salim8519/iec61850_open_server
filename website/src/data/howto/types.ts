// Search guides: one page per question engineers type into a search engine
// (/guides/<slug>/). Every step names what the real app shows; claims stay within the
// features page and the FAQ. Screenshots are real captures of the demo station only.
import type { ImageMetadata } from 'astro';
import { SITE } from '../site';

export interface HowToStep { title: string; text: string; image?: { src: ImageMetadata; alt: string }; }
export interface HowToProblem { q: string; a: string; }
export interface HowTo {
  slug: string;
  title: string;        // the page heading, phrased like the search
  short: string;        // card label on the guides list
  description: string;  // meta description, about 150 characters
  answer: string;       // the short answer under the heading
  needs: string[];
  steps: HowToStep[];
  problems: HowToProblem[];
  video: string;        // tutorial slug from data/tutorials.ts
  updated: string;      // ISO date of the last content change
}

export const guidePath = (guide: HowTo) => `/guides/${guide.slug}/`;

export const guideSchema = (guide: HowTo, image: string) => {
  const url = new URL(guidePath(guide), SITE.url).href;
  const publisher = { '@type': 'Organization', name: SITE.name, url: SITE.url };
  return {
    '@context': 'https://schema.org',
    '@graph': [
      { '@type': 'TechArticle', headline: guide.title, description: guide.description, url, image,
        datePublished: guide.updated, dateModified: guide.updated, author: publisher, publisher,
        about: 'IEC 61850' },
      { '@type': 'BreadcrumbList', itemListElement: [
        { '@type': 'ListItem', position: 1, name: 'Home', item: SITE.url },
        { '@type': 'ListItem', position: 2, name: 'Guides', item: new URL('/guides/', SITE.url).href },
        { '@type': 'ListItem', position: 3, name: guide.short, item: url },
      ] },
    ],
  };
};
