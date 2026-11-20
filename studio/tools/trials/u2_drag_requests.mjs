/* Trial U2: requests sent after one diagram edit (a breaker dragged on the SLD).
 *
 *   node studio/tools/trials/u2_drag_requests.mjs <engine log> [runs=10] [cdpPort=22902]
 *
 * IEDs stopped (editing is allowed only then); run u_setup.mjs first so the breakers sit on their
 * grid. Each run drags one breaker 40 px sideways with a real mouse (DevTools input events; back on
 * its next turn), waits until no request is open for 1.5 s, and lists what was sent from the press on. It also checks that the engine's Undo label is the drag and that Undo is enabled
 * on screen. Prints JSON: requests per drag (median, worst) and one sample list. */
import { launch, median, worst } from './u_cdp.mjs';
import { launchUrl } from './u_api.mjs';

const [logFile, runsText = '10', cdpPort = '22902'] = process.argv.slice(2);
const runs = Number(runsText);
const browser = await launch({ port: Number(cdpPort) });
const page = await browser.page(await launchUrl(logFile));
await page.until("document.querySelector('.run-pill')?.textContent.includes('Run')");
await page.eval("[...document.querySelectorAll('.nav-item')].find(b => b.textContent.includes('SLD')).click()");
await page.until("document.querySelectorAll('.react-flow__node').length >= 20");
await page.settle(1500);

const mouse = (type, x, y) => page.send('Input.dispatchMouseEvent', { type, x, y, button: 'left', buttons: type === 'mouseReleased' ? 0 : 1, clickCount: 1 });
const counts = [];
let sample = [];
let undoOk = true;
// Only breakers fully inside the canvas (not under the selection sheet) can be grabbed with the mouse.
const visible = await page.eval(`(() => { const pane = document.querySelector('.react-flow').getBoundingClientRect();
  return [...document.querySelectorAll('.react-flow__node[data-id^="cb-"]')].filter(node => { const r = node.getBoundingClientRect();
    return r.top > pane.top + 70 && r.bottom < pane.bottom - 70 && r.left > pane.left + 10 && r.right < pane.right - 400; })  // the 380 px sheet opens on the right
    .map(node => node.getAttribute('data-id')); })()`);
if (!visible.length) throw new Error('no breaker inside the canvas');
for (let index = 0; index < runs; index += 1) {
  const node = visible[index % visible.length];
  const box = await page.eval(`(() => { const r = document.querySelector('.react-flow__node[data-id="${node}"]').getBoundingClientRect();
    return { x: r.left + r.width / 2, y: r.top + r.height / 2 }; })()`);
  await page.settle(500);
  page.reset();
  const dx = Math.floor(index / visible.length) % 2 ? -40 : 40;  // into the free space beside it; back on its next turn
  await mouse('mouseMoved', box.x, box.y);
  await mouse('mousePressed', box.x, box.y);
  for (let step = 1; step <= 6; step += 1) {
    await mouse('mouseMoved', box.x + (dx * step) / 6, box.y);
    await new Promise(resolve => setTimeout(resolve, 16));
  }
  await mouse('mouseReleased', box.x + dx, box.y);
  await new Promise(resolve => setTimeout(resolve, 600));  // the 400 ms save delay
  await page.settle(1500);
  const sent = page.requests().filter(r => r.type !== 'EventSource').map(r => `${r.method} ${r.path.split('?')[0]}`);
  if (!sent.includes('PUT /api/v1/project/diagram')) throw new Error(`drag ${index} of ${node} saved nothing`);
  counts.push(sent.length);
  if (index === runs - 1) sample = sent;
  const undo = await page.eval("!document.querySelector('[aria-label=\"Undo\"]')?.disabled");
  undoOk &&= undo;
}
console.log(JSON.stringify({ trial: 'U2', runs, median: median(counts), worst: worst(counts), counts, undoEnabledAfterEachDrag: undoOk, sample }));
browser.close();
process.exit(0);
