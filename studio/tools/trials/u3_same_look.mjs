/* Trial U3 check: lazy pages load their stylesheets later than before, so prove every place looks the
 * same in today's build and the trial build (1366x768): same elements, same boxes, same computed
 * colours, fonts, borders and spacing.
 *
 *   node studio/tools/trials/u3_same_look.mjs <today engine log> <trial engine log> [cdpPort=22907]
 *
 * Both instances hold the same u_setup.mjs project (IEDs stopped). The SLD drawing itself is
 * skipped (it keeps its own zoom), as in studio/tools/layout_snapshot.js. Prints JSON per place. */
import { launch } from './u_cdp.mjs';
import { launchUrl } from './u_api.mjs';

const [todayLog, trialLog, cdpPort = '22907'] = process.argv.slice(2);
const PLACES = ['Devices', 'SLD', 'Test', 'Tools', 'Settings', 'Devices'];
const STYLE = ['color', 'background-color', 'font-size', 'font-weight', 'font-family', 'line-height', 'border-top-color',
  'border-top-width', 'border-radius', 'padding-top', 'padding-left', 'margin-top', 'margin-left', 'display', 'gap',
  'box-shadow', 'opacity', 'text-align', 'cursor'];

const snapshot = `(() => {
  const round = v => Math.round(v * 10) / 10;
  const out = [];
  for (const element of document.querySelector('.app').querySelectorAll('*')) {
    if (element.closest('.react-flow__viewport')) continue;
    const style = getComputedStyle(element);
    if (style.display === 'contents' || style.display === 'none' || style.visibility === 'hidden') continue;
    const box = element.getBoundingClientRect();
    if (!box.width || !box.height) continue;
    const name = typeof element.className === 'string' ? element.className.trim().split(/\\s+/)[0] : '';
    out.push([element.tagName.toLowerCase() + (name ? '.' + name : ''), round(box.left), round(box.top), round(box.width),
      round(box.height), ${JSON.stringify(STYLE)}.map(key => style.getPropertyValue(key)).join('|')]);
  }
  return out;
})()`;

async function collect(log, port) {
  const browser = await launch({ port });
  const page = await browser.page(await launchUrl(log));
  await page.until("document.querySelectorAll('.list-row').length > 0");
  await page.settle(1000);
  const shots = [];
  for (const place of PLACES) {
    await page.eval(`[...document.querySelectorAll('.nav-item')].find(b => b.textContent.includes('${place}')).click()`);
    await page.settle(1200);
    await new Promise(resolve => setTimeout(resolve, 800));
    shots.push({ place, items: await page.eval(snapshot) });
  }
  browser.close();
  await new Promise(resolve => setTimeout(resolve, 1500));
  return shots;
}

const today = await collect(todayLog, Number(cdpPort));
const trial = await collect(trialLog, Number(cdpPort));
const report = today.map((shot, index) => {
  const other = trial[index].items;
  const diffs = [];
  if (shot.items.length !== other.length) diffs.push({ count: [shot.items.length, other.length] });
  for (let at = 0; at < Math.min(shot.items.length, other.length) && diffs.length < 8; at += 1) {
    const a = shot.items[at];
    const b = other[at];
    const off = a[0] !== b[0] || [1, 2, 3, 4].some(i => Math.abs(a[i] - b[i]) > 0.5) || a[5] !== b[5];
    if (off) diffs.push({ at, today: a, trial: b });
  }
  return { place: shot.place, elements: shot.items.length, same: diffs.length === 0, diffs };
});
console.log(JSON.stringify({ trial: 'U3-look', same: report.every(item => item.same), report }));
process.exit(0);
