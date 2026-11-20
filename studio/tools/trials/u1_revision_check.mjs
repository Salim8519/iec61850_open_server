/* Trial U1 evidence: the runtime-only actions leave the project unchanged (same revision, same
 * body apart from runtime overlays), so the browser has nothing to read again after them.
 *
 *   node studio/tools/trials/u1_revision_check.mjs <engine log>   (IEDs running, u_setup.mjs project)
 *
 * Prints one line per action: route, HTTP outcome, revision before/after, project body equal. */
import { launchUrl, session } from './u_api.mjs';

const { call } = await session(await launchUrl(process.argv[2]));
const strip = project => JSON.stringify({ ...project, startup_recovery: null });
const first = await call('/project');
const ied = first.ieds[0];
const point = { ied_id: ied.id, ref: `${ied.name}CTRL/GGIO1.SPCSO1.stVal` };
const actions = [
  ['POST /signals/update', () => call('/signals/update', 'POST', { ...point, value: true })],
  ['POST /signals/pulse', () => call('/signals/pulse', 'POST', { ...point, active_value: true, duration_ms: 200 })],
  ['POST /signals/release-override', () => call('/signals/release-override', 'POST', point)],
  ['POST /equipment/{node}/test', () => call('/equipment/cb-1/test', 'POST', { action: 'open' })],
  ['POST /equipment/{node}/operate', () => call('/equipment/cb-1/operate', 'POST', { action: 'close' })],
  ['POST /equipment/{node}/operate (tap form)', () => call('/equipment/cb-2/operate', 'POST', { action: 'open' })],
];
let all = true;
for (const [name, run] of actions) {
  const before = await call('/project');
  let outcome = 'ok';
  try { await run(); } catch (error) { outcome = `refused: ${String(error.message).slice(0, 80)}`; }
  await new Promise(resolve => setTimeout(resolve, 300));
  const after = await call('/project');
  const same = before.revision === after.revision && strip(before) === strip(after);
  all &&= same;
  console.log(JSON.stringify({ action: name, outcome, before: before.revision, after: after.revision, sameProject: same }));
}
console.log(JSON.stringify({ allUnchanged: all }));
