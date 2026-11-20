/* Trial U4 setup: a 5000-point IEC 104 gateway list on the u_setup.mjs project.
 *
 *   node studio/tools/trials/u4_gateway_setup.mjs <engine log> [total=5000]
 *
 * IEDs stopped. Proposes every point the 20 sample IEDs offer, fills up to `total` with own
 * (made-up) points so the table holds exactly `total` rows, and gives every measured IED point a
 * "Smooth wave" generator (sine, 10 s) so values change every second while the IEDs run. The
 * gateway itself is not started (its point table shows the IEDs' live values). Prints counts. */
import { launchUrl, session } from './u_api.mjs';

const [logFile, totalText = '5000'] = process.argv.slice(2);
const total = Number(totalText);
const { call } = await session(await launchUrl(logFile));
const protocol = 'iec104';
const proposed = await call('/tools/gateway/propose', 'POST', { protocol, mode: 'replace', current: [] });
let points = proposed.points.slice(0, total);
while (points.length < total) {  // the engine plans at most 1000 own points at a time
  const count = Math.min(1000, total - points.length);
  const plan = await call('/tools/gateway/points/new', 'POST', { protocol, kind: 'float', count,
    from: 20000 + points.length, name: `Spare${points.length}`, points });
  if (!plan.ok) throw new Error(`own points refused: ${JSON.stringify(plan.rows.find(row => row.problem))}`);
  points = [...points, ...plan.points];
}
const saved = await call('/tools/gateway/points', 'PUT', { protocol, points });
const measured = saved.points.filter(row => row.source !== 'manual' && row.kind === 'float' && row.ied && row.ref);
let generators = 0;
for (const row of measured) {
  try {
    await call('/signals/source', 'POST', { ied_id: row.ied, ref: row.ref,
      spec: { mode: 'sine', offset: 50, amplitude: 40, period: 10 } });
    generators += 1;
  } catch { /* a reference that takes no generator stays static */ }
}
console.log(JSON.stringify({ proposed: proposed.points.length, saved: saved.points.length, measured: measured.length, generators }));
