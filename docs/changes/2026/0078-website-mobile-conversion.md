# 0078: Website phone-first navigation, faster reading, value and buying trust

Date: 2026-10-17. Decision: [ADR-0053](../../decisions/ADR-0053-website-mobile-conversion.md).

Owner requests:
- make scadabench.com professional, quick to scroll, complete and fully phone-friendly, so
  engineers see the value and want to buy;
- add navigation for Home, Tutorials, the buy page and the other pages.

## Change

- **Navigation:**
  - The full nav (Home, Features, Tutorials, Pricing, Download, FAQ, Contact) appears in the
    header, together with Buy (live price) and Free trial.
  - Below 1080px there is a phone menu sheet. Before this change phones had no menu at all
    (`nav { display:none }`).
  - The footer gains Buy a license, Refund policy and the support address.
- **Phones:**
  - a bottom buy bar after the first screen, hidden over the page's own buy buttons;
  - a 3-bay hero poster with readable labels;
  - tap-to-enlarge screenshots in a lightbox;
  - stacked full-width buttons;
  - a tighter header at 360px.
- **Speed:**
  - quicker reveal and shorter Story steps;
  - `content-visibility` on lower sections;
  - hover/tap prefetch (the installer link opts out);
  - no per-frame blur on phones;
  - 720p `+faststart` tutorial copies (`marketing/videos/web_encode.py`), shipped by `deploy.cmd`.
- **Value:**
  - home: "See it working" demo video, a hardware-bench comparison, and a "Built by
    experienced SCADA and protection engineers" block;
  - tutorial lengths now match the real videos (6:19 in total).
- **Buying:**
  - a trust row (PayPal checkout, key on screen and by email, 7-day refund policy, offline
    after activation) on pricing, the closing panel and download;
  - "what happens after you pay";
  - a pricing FAQ from the shared `data/faqs.ts`;
  - the refund policy in EULA section 6 (`#refunds`) and the FAQ;
  - grouped FAQ with jump chips.
- **Fixes:**
  - The `$` and the price no longer split inside buttons (flex gap).
  - Inactive Story screenshots are out of the tab order.
  - The paper accent meets 4.5:1 contrast (`--accent-on-paper`).
  - `deploy.cmd` stopped uploading the raw `*.capture.mp4` files.

## Evidence

- `npm run build`: 11 pages, no errors.
- `python studio/tools/verify.py`: file lengths, architecture, documentation, history and
  specifications all PASS.
- **Lighthouse 12** (mobile emulation, headless Chrome, local build):

  | Page | Performance | Accessibility | SEO | LCP | CLS |
  |---|---|---|---|---|---|
  | home | 98 | 100 | 100 | 2.2 s | 0.005 |
  | pricing | 100 | 100 | 100 | 1.3 s | 0.001 |
  | faq | 100 | 100 | 100 | 1.3 s | 0 |
  | features | 100 | 100 | 100 | 1.3 s | 0 |
  | download | 100 | 100 | 100 | 1.7 s | 0 |
  | tutorials | 95 | 100 | 100 | 2.9 s | 0.001 |

  Best practices is 96 only because the live-price request is refused on localhost. The
  license server allows only scadabench.com.
- **Chrome at 360, 390 and 768px** (iframes) and on desktop:
  - no horizontal overflow;
  - the menu opens with all links and both buying buttons;
  - the buy bar shows after the hero and hides at Closing and over the pricing cards;
  - the lightbox opens a 1568px screenshot that scrolls sideways;
  - the FAQ jump chips land on the right group;
  - the comparison stacks into cards on phones.
- **Videos** (`web_encode.py`): 97.9 MB down to 16.2 MB for the seven, the largest 3.5 MB.
  Lengths were measured with ffprobe.

## Limitations

- Not published yet: `website/deploy.cmd` still has to be run, and the site checked on a
  real phone.
- The test Chrome window was throttled (2 to 4 fps on any page), so some screenshots were
  taken before reveal; the DOM state was confirmed by script.
- The license-server pages still use the old paper green.
- There is no code-signing certificate, so Windows SmartScreen still warns on download. That
  is the largest trust gap left.
- The refund is manual: a PayPal refund plus key deactivation by the owner.
