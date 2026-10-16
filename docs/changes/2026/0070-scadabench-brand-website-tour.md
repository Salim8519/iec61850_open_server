# 0070: ScadaBench brand, icon, website, guided tour and lazy server operations

Date: 2026-10-17. Owner requests: an identity and icon, an Apple-grade fast landing page with
3D, tutorials, Cloudflare deployment, Mailgun, and one-command server deploy.
Decision: [ADR-0047](../../decisions/ADR-0047-scadabench-brand-website-tour.md). Spec: [Tour](../../specs/tour.md).

## Change

- **Rename** to ScadaBench across the UI, launcher messages, packaging, installer, license
  server texts, the packaged README and tests. `launcher/legacy_folder.py` moves an old
  `%LOCALAPPDATA%\SubstationStudio` folder once.
- **Icon pipeline:** `studio/branding/icon.svg`, `tools/build_icons.py`. Favicons are in
  `frontend/public/assets/brand/`; the PyInstaller specs and the installer use `icon.ico`.
- **Website** `website/` (Astro):
  - home with an SVG poster, a lazy Three.js substation, a sticky scroll story with real
    screenshots, facts, a feature grid and a safe-by-design section;
  - features with keyword sections, pricing, download, tutorials, FAQ (FAQPage JSON-LD),
    contact, changelog, EULA draft, privacy, 404, sitemap, robots and an OG image;
  - `deploy.cmd`.
- **Cloudflare (owner's account, through the connector and wrangler):**
  - Pages project `scadabench` with the apex and `www` domains;
  - tunnel `scadabench-license` routing `license.scadabench.com` to `127.0.0.1:18090`, with DNS;
  - Email Routing for `support@` and `licenses@` to the owner's Gmail;
  - Mailgun DNS for `mg.scadabench.com` (SPF, DKIM 2048, MX, tracking CNAME, DMARC).
- **Guided tour:** `features/tour/`, `copy/tour.ts`, `data-tour` targets, a Help menu in the
  sidebar, and the `Menu` options `up`, `triggerClass` and `triggerTour`. `PreferenceStore` and
  `/api/v1/preferences` back it.
- **License server:**
  - `license-server set` (non-interactive settings);
  - `setup.sh` uses the binary built on Windows, applies `SET_ARGS`, and installs the dedicated
    `scadabench-tunnel` unit;
  - `deploy/push.ps1` and the root `deploy-server.cmd`;
  - `deploy/bootstrap-user.sh` creates the `scadabench` deploy login with limited sudo.

## Evidence

- Python: 298 passed (2 skipped: portable tests wait for a rebuilt `dist/ScadaBench`).
  Frontend: 95 tests plus 3 tour tests; `tsc` clean. Go: all packages pass, including the new
  reset-trial/delete store test. `set` was checked against a temporary data folder.
- Icons: 16, 32 and 180 px frames inspected; the ICO holds 9 sizes (45 KiB).
- Website build: 11 pages. Initial JS is 1.2 KB gzipped; the 3D chunk (135 KB gzipped) loads
  only after page load on capable desktops; HTML is 7 KB gzipped.
- Browser: desktop hero and 3D, scroll story (active step and screenshot match), phone width
  in a 390 px frame after fixing squeezed images.
- Live: `https://scadabench.com`, `www`, `/pricing/` and `/sitemap.xml` answer 200 in about
  0.55 s, and an unknown path answers 404. Both Pages domains are active.
- Tour in a running app: it opens on first start, rings Devices import, Build from SCD, Run and
  Help, and stays closed after a reload (`tour_seen_version` 1).
- Mailgun DNS resolves through 1.1.1.1 and 8.8.8.8; Mailgun still showed "Unverified" right
  after creation, before its next check.

## Limitations

- The license server is not deployed yet: the owner must authorise the deploy key on
  91.99.146.103 once. The real release key and the downloadable installer follow that.
- The tutorial videos are not rendered yet; the Tutorials page shows written steps and
  "Coming soon".
- The EULA is a draft for legal review. The refund policy is the owner's decision.
- Lighthouse was not run; speed is judged from bundle sizes and response times.
