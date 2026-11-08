# 0217: Website search guides

Owner: "how to rank the scadabench top", then "OK" to the first three guide pages
([ADR-0137](../../decisions/ADR-0137-website-search-guides.md)).

## Before

- scadabench.com was added to Google Search Console as a domain property, verified through
  Cloudflare (one TXT record, one-time authorization) and the sitemap was submitted.
- Google (personalisation off, 2026-11-08): #1-4 for "scadabench", #4 for "SCADA test bench";
  not in the top 20 for "IEC 61850 IED simulator" or "IEC 61850 simulator"; not on page 1 for
  "simulate IED from CID file", "IEC 61850 GOOSE simulator", "IEC 61850 server simulator SCD file",
  "IEC 61850 SCADA testing software" or "SCADA simulator".

## Change

| Area | Files |
| --- | --- |
| Guide data | `website/src/data/howto/` (`types.ts`, `index.ts`, `simulate-ied.ts`, `goose-test.ts`, `scada-test.ts`) |
| Pages | `website/src/pages/guides/index.astro`, `website/src/pages/guides/[slug].astro` |
| Body | `website/src/components/GuideBody.astro` |
| Links | `website/src/pages/sitemap.xml.ts`, `website/src/components/Footer.astro`, `website/src/pages/tutorials.astro` |
| Preview | `.claude/launch.json` `website-preview` (astro preview on port 4321, Node 22.23) |

New URLs:
- `/guides/`
- `/guides/simulate-iec-61850-ied-from-cid-scd/`
- `/guides/test-iec-61850-goose-without-test-set/`
- `/guides/test-scada-without-real-ieds/`

## Evidence

- `astro build` (Node 22.23.2): 15 pages; sitemap 14 URLs; titles "Simulate an IEC 61850 IED from a
  CID or SCD file | ScadaBench", "Test IEC 61850 GOOSE without a test set | ScadaBench", "Test SCADA
  without real IEDs | ScadaBench", "IEC 61850 guides | ScadaBench".
- Every UI label in the guides found in `studio/frontend/src` (Import CID / SCD, Monitor real
  device, Use a sample IED, Test points, Put my IEDs on a network, Where GOOSE goes, Also share with
  the network, GOOSE & Sampled Values traffic, SCADA gateway & master, Start gateway, Paste from
  Excel, Add all GOOSE IEDs).
- Preview at 1366x768 and 375x812: no sideways scroll, steps, screenshots (AVIF), video and
  TechArticle data present; guides list shows three cards.

## Published

- Pages deployment d23b34fb with the `deploy.cmd` steps (build, 0.1.33 installer, `_redirects`,
  tutorial videos). Skipped on purpose: the portable and libiconv R2 uploads (already there for
  0.1.33) and the release record step (0.1.33 already announced; it restarts the license server).
- Live installer SHA-256 `b58f1d27…71ca8d` identical before and after.
- Live: `/guides/`, the three guides, `/sitemap.xml` (14 URLs), `/tutorials/`, `/videos/goose-sld.mp4`
  and `/download/ScadaBench-Setup-0.1.33.exe` answer 200; the SCADA guide carries its title and
  TechArticle data; the home page footer links `/guides/`.
- Search Console: indexing requested for `/guides/` and the three guides ("URL was added to a
  priority crawl queue"); before that each was "URL is unknown to Google".

## Limitations

- Locally the tutorial video posters answer 404; the publish step copies the videos in.
- Screenshots reuse the four home-page captures; the import dialog, Where GOOSE goes and the
  gateway point list have no screenshot of their own yet.
- In this session's shell `node` is 22.11, which Astro 7.4 refuses; the build used
  `%LOCALAPPDATA%\pi-node\current` (22.23.2).
- Ranking effect unmeasured; Search Console data starts about a day after verification.
