import type { APIRoute } from 'astro';
import { SITE } from '../data/site';
import { HOWTOS, guidePath } from '../data/howto';
import { getCollection } from 'astro:content';
import { docPath, sortDocs } from '../data/docs';

const PAGES = ['/', '/features/', '/pricing/', '/download/', '/tutorials/', '/faq/', '/contact/', '/changelog/',
  '/legal/eula/', '/legal/privacy/', '/guides/', ...HOWTOS.map(guidePath)];

export const GET: APIRoute = async () => {
  const docs = sortDocs(await getCollection('docs')).map(docPath);
  const urls = [...PAGES, '/docs/', '/docs/all/', ...docs].map((path) => `  <url><loc>${new URL(path, SITE.url).href}</loc></url>`).join('\n');
  const body = `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls}\n</urlset>\n`;
  return new Response(body, { headers: { 'Content-Type': 'application/xml' } });
};
