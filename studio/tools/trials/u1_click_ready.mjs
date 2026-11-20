/* Trial U1: click-to-ready time of runtime-only actions on the ~20-IED project.
 *
 *   node studio/tools/trials/u1_click_ready.mjs <engine log> [runs=20] [action=value|test] [cdpPort=22901]
 *
 * The IEDs must run (u_setup.mjs, then POST /runtime/start). "value" presses True/False of a test
 * point on Test › Test points (POST /signals/update); "test" presses a breaker's Test button on the
 * SLD sheet (POST /equipment/{node}/test). Ready = the top-bar Stop button is enabled again (it is
 * disabled while an action runs), measured inside the page from the click. Also counts the
 * requests each press sends. Prints JSON: runs, median, worst and the requests per press. */
import { launch, median, worst } from './u_cdp.mjs';
import { launchUrl } from './u_api.mjs';

const [logFile, runsText = '20', action = 'value', cdpPort = '22901'] = process.argv.slice(2);
const runs = Number(runsText);
const browser = await launch({ port: Number(cdpPort) });
const page = await browser.page(await launchUrl(logFile));
await page.until("document.querySelector('.run-pill')?.textContent.includes('Stop')");
const place = action === 'value' ? 'Test' : 'SLD';
await page.eval(`[...document.querySelectorAll('.nav-item')].find(b => b.textContent.includes('${place}')).click()`);
await page.settle(1000);

// Observe the Stop button's disabled attribute; record when it last became enabled after the click.
await page.eval(`(() => {
  window.__u1 = { changes: [] };
  const pill = document.querySelector('.run-pill');
  new MutationObserver(() => window.__u1.changes.push([performance.now(), pill.disabled])).observe(pill, { attributes: true, attributeFilter: ['disabled'] });
  return true;
})()`);

async function press(index) {
  if (action === 'value') {
    const label = index % 2 ? 'False' : 'True';
    return page.eval(`(() => {
      const rows = [...document.querySelectorAll('button')].filter(b => b.textContent.trim() === '${label}');
      const button = rows[${index} % rows.length];
      window.__u1.changes = []; window.__u1.click = performance.now(); button.click(); return !!button;
    })()`);
  }
  // SLD: select breaker n (its sheet opens), then press the sheet's Test "Open" or "Closed" button.
  const node = `cb-${(index % 7) + 1}`;
  await page.eval(`document.querySelector('.react-flow__node[data-id="${node}"]')?.dispatchEvent(new MouseEvent('click', { bubbles: true }))`);
  await page.until(`document.querySelectorAll('.sheet .test-actions button').length >= 2`, 5000);
  return page.eval(`(() => {
    const word = ${index} % 2 ? 'Open' : 'Closed';
    const button = [...document.querySelectorAll('.sheet .test-actions button')].find(b => b.textContent.trim() === word);
    window.__u1.changes = []; window.__u1.click = performance.now(); button.click(); return !!button;
  })()`);
}

const times = [];
const counts = [];
for (let index = 0; index < runs; index += 1) {
  await page.settle(300);
  page.reset();
  if (!await press(index)) throw new Error('button not found');
  await page.until("window.__u1.changes.length > 0 && !document.querySelector('.run-pill').disabled", 15000);
  await page.settle(500);
  const result = await page.eval('({ click: window.__u1.click, changes: window.__u1.changes })');
  const enabled = result.changes.filter(([, disabled]) => !disabled).map(([at]) => at);
  times.push(Math.round(Math.max(...enabled) - result.click));
  counts.push(page.requests().filter(r => r.type !== 'EventSource').length);
  await new Promise(resolve => setTimeout(resolve, 150));
}
console.log(JSON.stringify({ trial: 'U1', action, runs, median: median(times), worst: worst(times), times,
  requestsPerPress: median(counts), sample: page.requests().map(r => `${r.method} ${r.path.split('?')[0]}`) }));
browser.close();
process.exit(0);
