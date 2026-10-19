# ADR-0071: Quiet, signed daily update check

Accepted: 2026-10-19. Change: [0105](../changes/2026/0105-ux-delete-picker-loading-goose-updates.md).
New contract: license server `GET /api/v1/release`; app `GET/PUT /api/v1/app/update`,
`POST /api/v1/app/update/check`. New outbound call from the app (besides licensing).

**Problem**

Installed copies never learn that a newer ScadaBench exists. The owner asked for an endpoint that
says so, protected against bots and floods, and explicitly said users must not be annoyed by it.

**Decision**

1. The license server serves the newest release at `GET /api/v1/release`: version, date, optional
   `min_version`, a one-line summary, the download page, and the installer/portable links with their
   SHA-256. Links may only point at `scadabench.com`, `www.scadabench.com` or
   `download.scadabench.com` (checked on the server and again in the app).
2. The answer is signed with the existing server Ed25519 key over
   `"scadabench-release-v1\n" + payload`. The prefix keeps release and license signatures apart:
   neither can ever pass as the other.
3. Abuse protection: the server signs once at start and every request only copies a ~1 KB byte
   string; per-IP token bucket (general class, 10/min by default) with `Retry-After`; the shared
   48-request in-flight cap; strong ETag with 304; `Cache-Control: public, max-age=300` so
   Cloudflare can absorb floods; no body, query or cookie is read; no CORS; nothing about the
   caller is stored. Origin stays hidden behind the Cloudflare tunnel.
4. The owner publishes with `owner.ps1 -Action release` (runs at the end of `website/deploy.cmd`):
   the record is built from `site.ts`, the changelog title and the SHA-256 of the built files, sent
   over the deploy SSH key as stdin to `license-server release`, then the service restarts.
5. The app checks 20–120 s after start and then at most once a day, with a 6 s timeout. It sends no
   machine ID, license, user or version (User-Agent "ScadaBench" only). Offline is stored as a quiet
   state, never shown as an error. The last good answer is kept.
6. Not annoying: no popup, sound or badge. One line above Settings ("Version 0.1.14 is ready") with
   × that hides it until the next version; Settings › About shows the details, Check now and a
   "once a day" switch (on by default, remembered). Only a release whose `min_version` is above the
   installed version is labelled "Important update" and cannot be hidden. Nothing is downloaded or
   installed by the app; the line opens the download page.
7. Development reports `0.1.13+dev` instead of `0.0.0+dev`, so the check behaves exactly as in the
   packaged app; a test keeps it equal to `website/src/data/site.ts`, and the protected build
   replaces any `x.y.z+dev`.

**Compatibility**

Old apps ignore the new endpoint. A server without a release record answers 404 `no_release`,
which the app shows as "did not answer". The license server must be redeployed before
`owner.ps1 -Action release` works; until then `deploy.cmd` prints a warning and the site still publishes.

**Rejected**

- A static `latest.json` on Pages: unsigned unless a second key is managed on the owner PC.
- Downloading and running the installer from the app: needs code signing and a tested upgrade path first.
- Proof of work on this endpoint: it is a cheap, cached read; rate limits and caching suffice.
