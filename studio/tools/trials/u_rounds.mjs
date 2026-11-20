/* Screen trials U1-U5: run one trial script against today's build and the trial build in
 * alternating rounds (today, trial, today, trial, ...), so both sides see the same PC state.
 *
 *   node studio/tools/trials/u_rounds.mjs <script.mjs> <rounds> <today engine log> <trial engine log> [script args...]
 *
 * Each engine instance serves its own frontend build (studio_server.exe beside its "frontend"
 * folder) on its own port and data folder. Prints one JSON line per round and a summary per side
 * with every numeric "median"/"worst" field the script printed. */
import { spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const [script, roundsText, todayLog, trialLog, ...rest] = process.argv.slice(2);
const here = dirname(fileURLToPath(import.meta.url));
const results = { today: [], trial: [] };
for (let round = 0; round < Number(roundsText); round += 1) {
  for (const [side, log] of [['today', todayLog], ['trial', trialLog]]) {
    const run = spawnSync(process.execPath, [join(here, script), log, ...rest], { encoding: 'utf8', timeout: 900000 });
    const line = run.stdout.trim().split('\n').pop() ?? '';
    let data = null;
    try { data = JSON.parse(line); } catch { data = { error: (run.stderr || line).slice(-400) }; }
    results[side].push(data);
    console.log(JSON.stringify({ round, side, ...data }));
  }
}
const summary = {};
for (const side of ['today', 'trial']) {
  summary[side] = {};
  for (const key of ['median', 'worst', 'fcpMedian', 'readyMedian', 'scriptMedian', 'requestsPerPress']) {
    const values = results[side].map(item => item?.[key]).filter(value => typeof value === 'number');
    if (values.length) summary[side][key] = values;
  }
}
console.log(JSON.stringify({ summary }));
