# ADR-0137: Website search guides

Accepted: 2026-11-08. Owner request: rank scadabench.com higher in search ("how to rank the
scadabench top"), then "OK" to writing the first three guide pages.

**Context**
- Search Console (verified 2026-11-08) shows the home and features pages indexed. A Google check
  the same day (personalisation off) found ScadaBench #1-4 for "scadabench" and #4 for "SCADA test
  bench", but outside the top 20 for "IEC 61850 IED simulator" and "IEC 61850 simulator", and absent
  for "simulate IED from CID file", "IEC 61850 GOOSE simulator" and "IEC 61850 SCADA testing
  software".
- The site had ten pages. Tutorials are anchors on one page, so no page answers one search
  question on its own URL.

**Decision**
- One page per search question at `/guides/<slug>/`, listed at `/guides/`. The data lives in
  `website/src/data/howto/` (one file per guide, `types.ts` for the shape and the schema), the
  page in `src/pages/guides/[slug].astro`, the body in `src/components/GuideBody.astro`.
- Each guide has the question as its heading, a short answer under it, what you need, numbered
  steps, the matching tutorial video, common problems and links to the other guides.
- Wording names only what the app shows (labels checked in `studio/frontend/src`) and stays within
  the features page and the FAQ. Screenshots are the existing real captures of the demo station
  (`src/assets/screens`); no customer or private SCD data.
- TechArticle and BreadcrumbList data per guide (no HowTo or FAQ rich-result markup, which search
  engines no longer show for this kind of site).
- Links: footer "Guides", a "Written guides" list on the tutorials page, and the sitemap. The header
  navigation is unchanged, so nothing new is clipped at 1366x768.
- First three guides: simulate an IED from a CID or SCD file, test GOOSE without a test set, test
  SCADA without real IEDs.

**Compatibility**
- New pages only; existing pages keep their URLs and content. `public/guides/` images and the
  `/guides/` pages share the folder without clashes.
- No new dependency. Shared tokens, `PageHeader`, `Closing` and the screenshot lightbox are reused.
