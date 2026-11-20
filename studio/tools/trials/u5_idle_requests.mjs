/* Trial U5: requests per minute while the user does nothing, page by page.
 *
 *   node studio/tools/trials/u5_idle_requests.mjs <engine log> [seconds=60] [pages=Devices,SLD,Test,Settings] [cdpPort=22903]
 *
 * A page name may carry one step after ">": "SLD>ied" (opens the IED component's
 * sheet, u_setup.mjs node ied-1), "Settings>wizard" (the network setup page, IEDs stopped) or
 * "Settings>labcard" (that page with the ScadaBench Lab card chosen: the lab status shows). Each
 * page is opened, left alone 5 s, then every
 * request is counted for `seconds`. Prints JSON per page: requests per minute and by path. */
import { launch } from './u_cdp.mjs';
import { launchUrl } from './u_api.mjs';

const [logFile, secondsText = '60', pagesText = 'Devices,SLD,Test,Settings', cdpPort = '22903'] = process.argv.slice(2);
const seconds = Number(secondsText);
const browser = await launch({ port: Number(cdpPort) });
const page = await browser.page(await launchUrl(logFile));
await page.until("!!document.querySelector('.run-pill')");
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
const clickText = (selector, text) => page.eval(`(() => { const b = [...document.querySelectorAll(${JSON.stringify(selector)})]
  .find(item => item.textContent.trim().startsWith(${JSON.stringify(text)})); b?.click(); return !!b; })()`);

const results = [];
for (const entry of pagesText.split(',')) {
  const [place, step] = entry.split('>');
  await clickText('.nav-item', place);
  await page.settle(800);
  if (step === 'wizard' || step === 'labcard') await clickText('button', 'Put my');
  if (step === 'labcard') {  // choose the ScadaBench Lab card: the page then shows the lab status (nothing is added yet)
    await page.until("!!document.querySelector('button[aria-label=\"Network card\"]')");
    let chosen = false;
    for (let tries = 0; tries < 30 && !chosen; tries += 1) {  // the card list fills in once Windows answered
      chosen = await page.eval(`(() => {
        const option = [...document.querySelectorAll('[role=option]')].find(o => o.textContent.includes('ScadaBench Lab'));
        if (option) { option.click(); return true; }
        const picker = document.querySelector('button[aria-label="Network card"]');
        if (picker.getAttribute('aria-expanded') !== 'true') picker.click();
        return false;
      })()`);
      if (!chosen) await pause(1000);
    }
    if (!chosen) throw new Error('the ScadaBench Lab card is not in the list');
  }
  if (step === 'ied') {
    await page.until("!!document.querySelector('.react-flow__node[data-id=\"ied-1\"]')");
    await page.eval(`document.querySelector('.react-flow__node[data-id="ied-1"]').dispatchEvent(new MouseEvent('click', { bubbles: true }))`);
  }
  await page.settle(800);
  await pause(5000);
  page.reset();
  await pause(seconds * 1000);
  const list = page.requests().filter(r => r.type !== 'EventSource');
  const byPath = {};
  for (const r of list) {
    const key = `${r.method} ${r.path.split('?')[0].replace(/[0-9a-f]{32}/g, '{id}')}`;
    byPath[key] = (byPath[key] ?? 0) + 1;
  }
  results.push({ page: entry, perMinute: Math.round((list.length * 60) / seconds), byPath });
}
console.log(JSON.stringify({ trial: 'U5', seconds, results }));
browser.close();
process.exit(0);
