/* Trial U2 check: with fewer requests, Undo/Redo, the auto save switch and a second tab still show
 * the right state (u_setup.mjs project, IEDs stopped).
 *
 *   node studio/tools/trials/u2_still_correct.mjs <engine log> [cdpPort=22908]
 *
 * Tab A drags breaker cb-1; tab B (a second tab) must show the new position and an enabled Undo
 * with the engine's label. Undo in A puts the breaker back in both tabs and enables Redo. Then the
 * File menu turns auto save off; a drag shows "Not saved · Ctrl+S" in both tabs, Ctrl+S in A shows
 * "Auto save off" in both, and turning it on again shows "All changes saved". Prints JSON checks. */
import { launch } from './u_cdp.mjs';
import { launchUrl } from './u_api.mjs';

const [logFile, cdpPort = '22908'] = process.argv.slice(2);
const pause = ms => new Promise(resolve => setTimeout(resolve, ms));
const browser = await launch({ port: Number(cdpPort) });
const url = await launchUrl(logFile);
const a = await browser.page(url);
await a.until("document.querySelectorAll('.list-row').length > 0");
const b = await browser.page(url.split('#')[0]);
await b.until("document.querySelectorAll('.list-row').length > 0");
for (const tab of [a, b]) {
  await tab.eval("[...document.querySelectorAll('.nav-item')].find(x => x.textContent.includes('SLD')).click()");
  await tab.until("!!document.querySelector('.react-flow__node[data-id=\"cb-1\"]')");
}
await pause(1500);
// A background tab is hidden: by design it subscribes to nothing until it shows again (push client).
// So the person "looks at" a tab first: bring it to the front, give it a moment, then read it.
let front = null;
const look = async tab => { if (front !== tab) { await tab.send('Page.bringToFront'); front = tab; await pause(1200); } return tab; };
const nodeX = async tab => (await look(tab)).eval("document.querySelector('.react-flow__node[data-id=\"cb-1\"]').style.transform");
const words = async tab => (await look(tab)).eval("document.querySelector('.topbar')?.textContent ?? ''");
const undo = async tab => (await look(tab)).eval("({ undo: !document.querySelector('[aria-label=\"Undo\"]').disabled, redo: !document.querySelector('[aria-label=\"Redo\"]').disabled, tip: document.querySelector('[aria-label=\"Undo\"]').title })");
async function drag(tab) {
  await look(tab);
  const box = await tab.eval(`(() => { const r = document.querySelector('.react-flow__node[data-id="cb-1"]').getBoundingClientRect();
    return { x: r.left + r.width / 2, y: r.top + r.height / 2 }; })()`);
  const mouse = (type, x, y) => tab.send('Input.dispatchMouseEvent', { type, x, y, button: 'left', buttons: type === 'mouseReleased' ? 0 : 1, clickCount: 1 });
  await mouse('mouseMoved', box.x, box.y);
  await mouse('mousePressed', box.x, box.y);
  for (let step = 1; step <= 6; step += 1) { await mouse('mouseMoved', box.x + step * 7, box.y); await pause(16); }
  await mouse('mouseReleased', box.x + 42, box.y);
  await pause(1500);
}
async function fileMenu(tab, label) {
  await (await look(tab)).eval("[...document.querySelectorAll('button')].find(x => x.textContent.trim() === 'File').click()");
  await pause(300);
  await tab.eval(`[...document.querySelectorAll('[role=menuitem], button')].find(x => x.textContent.trim() === ${JSON.stringify(label)}).click()`);
  await pause(1500);
}
const checks = {};
const before = await nodeX(a);
await drag(a);
checks.movedInA = (await nodeX(a)) !== before;
checks.movedInB = (await nodeX(b)) === (await nodeX(a));
const undoB = await undo(b);
checks.undoEnabledInBoth = (await undo(a)).undo && undoB.undo;
checks.undoTip = undoB.tip;
await (await look(a)).eval("document.querySelector('[aria-label=\"Undo\"]').click()");
await pause(1500);
checks.positions = `${before} -> ${await nodeX(a)}`;
checks.undoPutBackA = (await nodeX(a)) === before;
checks.undoPutBackB = (await nodeX(b)) === before;
checks.redoEnabledInBoth = (await undo(a)).redo && (await undo(b)).redo;
await fileMenu(a, 'Turn off auto save');
checks.offShownInB = (await words(b)).includes('Auto save off');
await drag(a);
checks.unsavedA = (await words(a)).includes('Not saved · Ctrl+S');
checks.unsavedB = (await words(b)).includes('Not saved · Ctrl+S');
await look(a);
await a.send('Input.dispatchKeyEvent', { type: 'keyDown', key: 's', code: 'KeyS', modifiers: 2, windowsVirtualKeyCode: 83 });
await a.send('Input.dispatchKeyEvent', { type: 'keyUp', key: 's', code: 'KeyS', modifiers: 2, windowsVirtualKeyCode: 83 });
await pause(1500);
checks.savedA = (await words(a)).includes('Auto save off') && !(await words(a)).includes('Not saved');
checks.savedB = (await words(b)).includes('Auto save off') && !(await words(b)).includes('Not saved');
await fileMenu(a, 'Turn on auto save');
checks.onA = (await words(a)).includes('All changes saved');
checks.onB = (await words(b)).includes('All changes saved');
const requests = a.requests().filter(r => r.type !== 'EventSource').map(r => `${r.method} ${r.path.split('?')[0]}`);
console.log(JSON.stringify({ trial: 'U2-correct', ok: Object.values(checks).every(v => v === true || typeof v === 'string'), checks,
  tabARequests: requests.length }));
browser.close();
process.exit(0);
