/* Trial U3: time to first paint and to a usable Devices page, plus the JavaScript the page loads.
 *
 *   node studio/tools/trials/u3_first_paint.mjs <engine log> [runs=10] [cdpPort=22905]
 *
 * Each run opens a fresh tab with the browser cache off (a first start), and records
 * first-contentful-paint, the time the Devices list shows the project (".device-list" rows), and
 * the bytes of script the page fetched before that. Then it opens every place once and reports
 * how long each switch took to show its page (the owner's "no flash" check is separate: a person
 * looks at 1366x768). Prints JSON with medians and worst runs. */
import { launch, median, worst } from './u_cdp.mjs';
import { launchUrl } from './u_api.mjs';

const [logFile, runsText = '10', cdpPort = '22905'] = process.argv.slice(2);
const launchAddress = await launchUrl(logFile);
const url = launchAddress.split('#')[0];
const browser = await launch({ port: Number(cdpPort) });
const signIn = await browser.page(launchAddress);  // the session cookie, shared by the later tabs
await signIn.until("document.querySelectorAll('.list-row').length > 0");
signIn.close();
console.error('signed in');
const paints = [];
const readies = [];
const scripts = [];
let switches = null;
for (let run = 0; run < Number(runsText); run += 1) {
  const page = await browser.page();
  await page.send('Network.setCacheDisabled', { cacheDisabled: true });
  // The moment the first Devices row exists, seen by an observer that starts with the document.
  await page.send('Page.addScriptToEvaluateOnNewDocument', { source: `new MutationObserver((_, observer) => {
    if (document.querySelector('.list-row')) { window.__u3ready = performance.now(); observer.disconnect(); }
  }).observe(document, { childList: true, subtree: true });` });
  await page.go(url);
  await page.until("window.__u3ready > 0", 30000);
  const timing = await page.eval(`(() => {
    const fcp = performance.getEntriesByName('first-contentful-paint')[0]?.startTime ?? null;
    const js = performance.getEntriesByType('resource').filter(e => e.name.includes('/assets/') && e.name.endsWith('.js')
      && e.startTime < window.__u3ready).reduce((sum, e) => sum + (e.encodedBodySize || e.transferSize || 0), 0);
    return { fcp, ready: window.__u3ready, js };
  })()`);
  console.error('run', run, JSON.stringify(timing));
  paints.push(Math.round(timing.fcp));
  readies.push(Math.round(timing.ready));
  scripts.push(timing.js);
  if (run === Number(runsText) - 1) {
    switches = {};
    for (const place of ['SLD', 'Test', 'Tools', 'Settings', 'Devices']) {
      const ms = await page.eval(`new Promise(resolve => {
        const before = document.querySelector('.topbar-title h1')?.textContent;
        const start = performance.now();
        [...document.querySelectorAll('.nav-item')].find(b => b.textContent.includes('${place}')).click();
        // Shown: the top bar title changed (a page switch commits the title and the page together).
        const check = () => document.querySelector('.topbar-title h1')?.textContent !== before
          ? resolve(Math.round(performance.now() - start)) : requestAnimationFrame(check);
        requestAnimationFrame(check);
      })`);
      switches[place] = ms;
      await page.settle(600);
    }
  }
  page.send('Page.close').catch(() => {});  // no answer comes back from a closed tab
  await new Promise(resolve => setTimeout(resolve, 300));
  page.close();
}
console.log(JSON.stringify({ trial: 'U3', runs: paints.length, fcpMedian: median(paints), fcpWorst: worst(paints), paints,
  readyMedian: median(readies), readyWorst: worst(readies), readies, scriptBytesBeforeReady: median(scripts), firstSwitchMs: switches }));
browser.close();
process.exit(0);
