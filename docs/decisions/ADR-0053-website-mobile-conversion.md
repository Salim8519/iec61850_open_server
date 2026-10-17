# ADR-0053: Website built phone-first, with fast reading and clear reasons to buy

Accepted: 2026-10-17. Owner request: make scadabench.com look professional and scroll fast.
It should give all the information, work completely on phones, and make engineers see the
value and want to buy. Navigation must reach Home, Tutorials, the buy page and every other
page.

**Decision**
- **Navigation everywhere.** `NAV` in `website/src/data/site.ts` lists Home, Features,
  Tutorials, Pricing, Download, FAQ and Contact.
  - From 1080px the header shows the list, plus Buy (live price) and Free trial.
  - Below 1080px a `<details>` menu sheet (`MobileMenu.astro`) shows the same list and both
    buying buttons. It opens without JavaScript. The script adds Esc, outside tap, close on
    link and a scroll lock.
  - The header blur applies only from 1080px. A backdrop filter would trap the fixed sheet
    inside the header, and it costs frames on phone GPUs.
- **Phone buy bar** (`StickyCta.astro`, `scripts/stickyCta.ts`):
  - It shows below 760px once the first screen has been scrolled away.
  - It hides while the page's own buying buttons (`[data-cta-zone]`) or the footer are on
    screen, so it never doubles them.
- **Readable on phones:**
  - a 3-bay hero poster with 24px labels;
  - every product screenshot opens in a native `<dialog>` lightbox, wider than the screen and
    scrollable sideways (`scripts/lightbox.ts`);
  - buttons stack full width.
- **Faster:**
  - Reveal animation shortened from 0.8s/28px to 0.45s/12px.
  - Story steps shortened from 64vh to 48vh.
  - `content-visibility:auto` on lower sections.
  - Astro hover/tap prefetch. Viewport prefetch was rejected because it would download the
    25 MB installer link; that link opts out.
  - Tutorial videos are shipped as 720p `+faststart` copies (`marketing/videos/web_encode.py`),
    98 MB down to 16 MB for the seven.
- **Value and trust, with no invented claims:**
  - a home demo video;
  - a hardware-bench comparison (no competitor names or prices);
  - a "Built by experienced SCADA and protection engineers" block, in the owner's own words,
    with no names or numbers;
  - a trust row next to every buy button;
  - "what happens after you pay";
  - a pricing FAQ.
  - One shared question list (`data/faqs.ts`) feeds the FAQ page, the pricing page and the
    structured data.
  - Tutorial lengths are the measured video lengths. The old "1 min" and "ten minutes" were
    wrong.
- **Refund policy, published plainly.**
  - The owner asked for a refund within a week, but not on request alone, and suggested a
    hidden agreement. Hidden terms mislead buyers and are generally unenforceable, so the
    conditions are visible instead.
  - The policy: a full PayPal refund within 7 days of purchase if ScadaBench does not work
    with the buyer's files or PC and support cannot fix it, and the key is deactivated. It
    asks buyers to use the free trial first.
  - It appears in the EULA (section 6, `#refunds`), the FAQ, pricing and the trust row.
    `SITE.refundDays` holds the number.
- **Contrast:** a new token `--accent-on-paper: #087a5c` (4.5:1 or better on every paper
  surface) replaces `#0b8f6c` for small green text on light sections.

**Compatibility**
- No new dependencies. Static output and the deploy command are unchanged.
- `deploy.cmd` prefers `out/web/` videos, falls back to the masters, and no longer uploads the
  raw `*.capture.mp4` files.
- The license-server pages still use `#0b8f6c` (`license-server/internal/api/pages/style.css`).
  Aligning them needs a server deploy and is left for the next server release.
- The refund policy is a business commitment. The owner refunds through PayPal and deactivates
  the key by hand; no automatic refund exists.
