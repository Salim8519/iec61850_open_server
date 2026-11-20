/* Screen trials U1-U5: build the ~20-IED loopback project in a temp engine instance.
 *
 *   node studio/tools/trials/u_setup.mjs <engine log file> [ieds=20] [portBase=22100]
 *
 * Adds sample IEDs Feeder01..FeederNN (the built-in template, no customer data), gives each its
 * own MMS port (portBase + n) on its private 127.50.0.x loopback address so nothing collides with
 * other instances, draws one breaker per IED bound to <IED>CTRL/XCBR1.Pos (state and command), and
 * adds one test point per IED (GGIO1.SPCSO1). Loopback mode only: no Lab card, no adapter. */
import { session, launchUrl } from './u_api.mjs';

const [logFile, count = '20', base = '22100'] = process.argv.slice(2);
const { call } = await session(await launchUrl(logFile));
const total = Number(count);
const name = n => `Feeder${String(n).padStart(2, '0')}`;

const prefs = await call('/preferences');
await call('/preferences', 'PUT', { tour_seen_version: prefs.tour_version });  // no tour over the pages
let project = await call('/project');
if (project.runtime !== 'stopped') await call('/runtime/stop', 'POST');
for (let n = project.ieds.length + 1; n <= total; n += 1) await call('/ieds/template', 'POST', { name: name(n) });
project = await call('/project');
const ieds = project.ieds.slice(0, total);
for (const [index, ied] of ieds.entries()) {
  await call(`/ieds/${ied.id}/settings`, 'PUT', { port: Number(base) + index + 1 });
}
project = await call('/project');

const nodes = ieds.map((ied, index) => {
  const ref = `${ied.name}CTRL/XCBR1.Pos`;
  return { id: `cb-${index + 1}`, type: 'equipment', position: { x: 120 + (index % 10) * 140, y: 120 + Math.floor(index / 10) * 220 },
    data: { label: `CB ${ied.name}`, kind: 'CB', rotation: 0,
      state_mapping: { mode: 'direct', source: { ied_id: ied.id, report_ref: null, ref: `${ref}.stVal` } },
      control_binding: { ied_id: ied.id, ref } } };
});
// One IED component (ADR-0112): its sheet lists the IED's GOOSE messages while the IEDs run (trial U5).
nodes.push({ id: 'ied-1', type: 'equipment', position: { x: 120, y: 600 },
  data: { label: ieds[0].name, kind: 'IED', rotation: 0, ied_id: ieds[0].id } });
const answer = await call('/project/diagram', 'PUT', { diagram: { nodes, edges: [] }, revision: project.revision, tab_id: 'trial-setup' });
const panel = ieds.map((ied, index) => ({ id: `tp-${index + 1}`, label: `Spare ${ied.name}`, ied_id: ied.id,
  ref: `${ied.name}CTRL/GGIO1.SPCSO1.stVal`, report_ref: null, duration_ms: 500, active_value: true }));
await call('/project/test-panel', 'PUT', panel);
project = await call('/project');
const bytes = JSON.stringify(project).length;
console.log(JSON.stringify({ ieds: project.ieds.length, nodes: project.diagram.nodes.length, tests: project.test_panel.length,
  revision: answer.revision, projectBytes: bytes, ports: project.ieds.map(ied => ied.port ?? ied.settings?.port).slice(0, 3) }));
