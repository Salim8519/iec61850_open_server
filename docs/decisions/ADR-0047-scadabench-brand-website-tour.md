# ADR-0047: ScadaBench brand, website and guided tour

Accepted: 2026-10-17. Owner request: a real identity (name, icon), a landing page that attracts
customers and opens fast, a beginner tour in the app, and lazy deployment.

**Decision**
- **Name: ScadaBench** (`scadabench.com`, owned on Cloudflare). Visible names change everywhere:
  - app title and sidebar;
  - EXE `ScadaBench.exe`;
  - installer (new AppId, per-user install);
  - `%LOCALAPPDATA%\ScadaBench`, with a one-time move of an existing `SubstationStudio` folder
    (`launcher/legacy_folder.py`; nothing is merged or deleted when both exist);
  - license server texts.

  Internal identifiers stay: `studio_app`, `.subsim`, the HWID prefix, `SUBS-` keys and the
  firewall rule names and group, because the network journal and cleanup depend on them.
- **Icon:** one master `studio/branding/icon.svg` ("S-Line": an S drawn as single-line wiring
  through a breaker). `tools/build_icons.py` renders it with headless Edge, resizes it with
  Windows System.Drawing and writes a PNG-in-ICO itself, so no new dependency is needed. The
  same assets are used by the dev browser tab, the frozen EXE, the installer, the Edge app
  window and the website.
- **Website:** `website/` is a static Astro site on Cloudflare Pages (project `scadabench`).
  - Speed: an SVG poster shows instantly, and the procedural Three.js substation (no model
    files) loads lazily, only on capable desktops. Initial JS is about 1 KB gzipped.
  - Screenshots are real captures of the app.
  - Claims are limited to evidenced features; IEC 60870-5-104 is shown only as "roadmap".
  - Deploy: `website/deploy.cmd` (wrangler). Domains, DNS, tunnel and email routing are
    managed through the owner's Cloudflare connection.
- **Guided tour:** `features/tour/` spotlight overlay, wording in `copy/tour.ts`, targets
  marked with `data-tour`. It opens on first start and from Help.
  - "Seen" is stored by the backend in `data/preferences.json` (`GET/PUT /api/v1/preferences`),
    because the packaged port can change.
  - Dev and EXE behave the same.
- **License server operations:**
  - `deploy-server.cmd` cross-compiles the Go server on Windows and uploads it.
  - `setup.sh` installs it without compiling on the server, applies settings with
    `license-server set`, and runs a dedicated `scadabench-tunnel` cloudflared unit, so other
    software on a shared host is untouched.
  - The tunnel token lives in `/etc/scadabench/tunnel.env` (root, 0600) and on Windows in
    `%LOCALAPPDATA%\ScadaBench-deploy` (user-only ACL), never in the repository.
- **Email:**
  - Cloudflare Email Routing forwards `support@` and `licenses@` to the owner's mailbox.
  - Mailgun sends license email from `mg.scadabench.com` (DNS managed in Cloudflare).
  - The owner types API keys and secrets into the server setup; they are never stored by tools
    or shown in chat.

**Compatibility**
- Development behaviour is unchanged apart from the name, the icon, the tour on first start and
  Help.
- Packaged builds use the new folder and migrate the old one once.
- New dependencies: Astro 7.4, Three.js 0.186 and Geist fonts (website only), and Wrangler
  (deploy tool, fetched by npx).
- Remotion videos are planned; a company license is needed above three employees.
