/* Trial U4: browser CPU of the 5000-point gateway point table while values change.
 *
 *   node studio/tools/trials/u4_point_table_cpu.mjs <engine log> [seconds=30] [rounds=5] [cdpPort=22904] [shot.png]
 *
 * u4_gateway_setup.mjs list, IEDs running (20 points with a sine generator change every second).
 * Opens Tools › SCADA gateway › Points and, per round, reads the page's main-thread busy time
 * (DevTools Performance metrics TaskDuration and ScriptDuration) over `seconds`. Also records the
 * rows and counts the table shows, so both sides can be compared. Prints JSON: ms of main-thread
 * work per second per round, median and worst. */
import { launch, median, worst } from './u_cdp.mjs';
import { launchUrl } from './u_api.mjs';

const [logFile, secondsText = '30', roundsText = '5', cdpPort = '22904', shot] = process.argv.slice(2);
const browser = await launch({ port: Number(cdpPort) });
const page = await browser.page(await launchUrl(logFile));
await page.until("document.querySelector('.run-pill')?.textContent.includes('Stop')");
const clickText = (selector, text) => page.eval(`(() => { const b = [...document.querySelectorAll(${JSON.stringify(selector)})]
  .find(item => item.textContent.trim().includes(${JSON.stringify(text)})); b?.click(); return !!b; })()`);
await clickText('.nav-item', 'Tools');
await page.settle(800);
await clickText('button, a, [role="button"]', 'SCADA gateway');
await page.settle(800);
await clickText('[role="tab"], .tabbar button, .segmented-control button', 'Points');
await page.until("document.querySelectorAll('table.gateway-points tbody tr').length >= 100", 20000);
await page.settle(1500);
if (shot) await page.screenshot(shot);

const shown = () => page.eval(`({ rows: document.querySelectorAll('table.gateway-points tbody tr').length,
  filter: document.querySelector('.filter-bar, [aria-label="Filter points"]')?.textContent.replace(/\\s+/g, ' ').slice(0, 120) ?? '',
  first: [...document.querySelectorAll('table.gateway-points tbody tr')].slice(0, 3).map(tr => tr.textContent.replace(/\\s+/g, ' ').slice(0, 80)) })`);
const perSecond = [];
const scripts = [];
for (let round = 0; round < Number(roundsText); round += 1) {
  const before = await page.metrics();
  await new Promise(resolve => setTimeout(resolve, Number(secondsText) * 1000));
  const after = await page.metrics();
  perSecond.push(Math.round(((after.TaskDuration - before.TaskDuration) * 1000) / Number(secondsText) * 10) / 10);
  scripts.push(Math.round(((after.ScriptDuration - before.ScriptDuration) * 1000) / Number(secondsText) * 10) / 10);
}
const frames = page.requests().filter(r => r.type !== 'EventSource').length;
console.log(JSON.stringify({ trial: 'U4', seconds: Number(secondsText), taskMsPerSecond: perSecond, median: median(perSecond),
  worst: worst(perSecond), scriptMsPerSecond: scripts, scriptMedian: median(scripts), httpRequests: frames, shown: await shown() }));
browser.close();
process.exit(0);
