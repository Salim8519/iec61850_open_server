// Online user documentation: one Markdown file per page under src/content/docs/<section>/<page>.md.
// The frontmatter is checked here, so a typo fails the build instead of showing a broken page.
import { defineCollection, z } from 'astro:content';
import { glob } from 'astro/loaders';
import { DOC_SECTION_IDS } from './data/docs';

const docs = defineCollection({
  loader: glob({ pattern: '**/*.md', base: './src/content/docs' }),
  schema: z.object({
    title: z.string().min(3),
    description: z.string().min(20),
    section: z.enum(DOC_SECTION_IDS),
    order: z.number().int().positive(),
    video: z.string().optional(),
  }),
});

export const collections = { docs };
