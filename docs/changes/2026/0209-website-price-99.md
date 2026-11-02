# 0209: Website built with the $99 price

Owner: "the website shows 149$ change it to 99$". The license server price was already 99.00
(set 2026-10-28 with the Lemon Squeezy store), and the page script that reads the live price
switched the buttons to 99 after loading. The pages themselves were still built with 149, so
visitors saw 149 for a moment, and the page description and the search-engine offer data said
149 USD.

## Change

| Area | Files |
| --- | --- |
| Website data | `website/src/data/site.ts`: `price: '99'` (buttons, pricing card, page description, schema.org offers) |

No license server, product, installer or portable change. The live-price script stays: a later
price change on the server still shows on the site without a rebuild.

## Evidence

- `npm run build`: 11 pages.
- Published with the `website/deploy.cmd` steps (Pages deployment 85edd0f8, same 0.1.31 installer
  and portable uploaded again). The release record step was skipped: 0.1.31 was already published
  and that step restarts the license server.
- Live `https://scadabench.com/pricing/` HTML: every price element is 99, the description reads
  "costs 99 USD", the offer data has `"price":"99"`, and "149" is nowhere in the page. The home page
  offer data also reads 99.
- `license.scadabench.com/api/v1/product`: `{"amount":"$99.00","price":"99.00",...}`.
- `/download/ScadaBench-Setup-0.1.31.exe` still answers 200.

## Limitations

- Search engines and link previews show 99 only after they fetch the pages again.
- The license server's built-in default (`license-server/internal/config/config.go`) is still
  149.00; it applies only to a server whose saved settings have no price. The live server has 99.00.
