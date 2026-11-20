# 0228 Online user manual on scadabench.com/docs, printable as one PDF

Date: 2026-11-20. Owner: "should we do manual pdf? Or docs system! Maybe docs online", then
"let haiku do the docs and you check his work quickly, send 2 haiku agents". Layer: website
(Astro, `website/`) only. No app code, no new dependency (Astro content collections are built in).

## What changed

- **Docs collection.** `website/src/content.config.ts` validates each page's frontmatter (title,
  description, section, order, optional tutorial `video` slug). A wrong section or an unknown
  video slug fails the build instead of shipping a broken page.
- **41 pages in 8 sections** under `website/src/content/docs/`: getting started, devices, SLD,
  test, tools, network, licence and VMs, troubleshooting. Plain English, task-first, button names
  in bold exactly as the app shows them; a page with a tutorial shows the video above the text.
- **Pages.** `/docs/` (sections), `/docs/<section>/<page>/` (sidebar, previous/next; the page list
  folds into a disclosure on phones), `/docs/all/` (the whole manual on one page with
  "Download as PDF", which prints with the site chrome hidden and a page break per page).
- **Site.** "Docs" in the header, phone menu and footer (`data/site.ts`); every docs page in the
  sitemap; `Prose.astro` gained an optional `aside` slot (other pages unchanged).

## Review (lead, after the two writers)

- Wording checked against the app source for the claims most likely to drift: trial length
  (7 days, same as the tutorials and how-to pages), download file names (`site.ts`), generator
  modes and "Resume generator" (`features/test`), Stop takes the Lab addresses off the card
  (owner decision N10 off, change 0222/ADR-0141 note), VM steps and firewall wording.
- Public-material rules: only the OpenSCD-based demo station and the sample IED are named; no
  private station names, no antivirus vendors, nothing that describes a way around the licence.
- `fnm exec --using=24 -- node node_modules/astro/bin/astro.mjs build`: 58 pages, no warnings
  from the docs. Every new file is under 300 lines.

## Limitations

- The print-to-PDF output was not checked in a print preview.
- Pages were written from the source and the tutorials, not by clicking through every step in the
  packaged app; the writers' open questions (converter upstream choice, update install flow,
  recovery notice wording) were left out of the pages rather than guessed.
- Not deployed: the site goes live with the next `website/deploy.cmd`, after the owner's OK.
