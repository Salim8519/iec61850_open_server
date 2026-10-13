import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import test from 'node:test';
import { HINTS } from '../../frontend/src/copy/hints.ts';
import { GLOSSARY } from '../../frontend/src/copy/glossary.ts';
import { COMPARE } from '../../frontend/src/copy/compare.ts';
import { GOOSE_STATES, LIVE_STATES, PROBLEMS, TONES, UI_STATUS, type Status } from '../../frontend/src/copy/status.ts';

const ROOT = new URL('../../', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1');
const read = (path: string) => readFileSync(join(ROOT, path), 'utf8');
const guided = ['frontend/src/features/network', 'frontend/src/features/devices', 'frontend/src/features/test',
  'frontend/src/features/settings', 'frontend/src/features/diagram', 'frontend/src/app', 'frontend/src/ui']
  .flatMap(folder => readdirSync(join(ROOT, folder)).filter(name => name.endsWith('.tsx'))
    .map(name => ({ name, text: read(`${folder}/${name}`) })));
const codes = (path: string, pattern: RegExp) => [...read(path).matchAll(pattern)].map(match => match[1]);

test('every referenced hint exists', () => {
  const used = new Set<string>();
  for (const file of guided) {
    for (const match of file.text.matchAll(/['"]((?:network|goose|live)\.[a-z_.]+)['"]/g)) used.add(match[1]);
  }
  for (const role of ['simulated', 'real', 'off']) used.add(`live.role.${role}`);
  for (const access of ['auto_reports', 'polling']) used.add(`live.access.${access}`);
  assert.ok(used.size > 30);
  for (const key of used) assert.ok(HINTS[key]?.length > 20, `Missing hint ${key}`);
});

test('glossary entries are plain sentences', () => {
  for (const [key, entry] of Object.entries(GLOSSARY)) {
    assert.match(entry.plain, /^[A-Z].{15,200}\.$/, `Glossary ${key} must be one plain sentence`);
  }
  for (const file of guided) {
    for (const match of file.text.matchAll(/term="([a-z]+)"/g)) assert.ok(GLOSSARY[match[1]], `${file.name}: unknown term ${match[1]}`);
  }
});

test('status tones never rely on colour', () => {
  const icons = new Set(Object.values(TONES).map(style => style.icon));
  assert.equal(icons.size, Object.keys(TONES).length, 'Each tone needs its own icon');
  for (const style of Object.values(TONES)) assert.ok(style.word && style.pattern);
  const all: Status[] = [...Object.values(GOOSE_STATES), ...Object.values(PROBLEMS), ...Object.values(LIVE_STATES),
    ...Object.values(UI_STATUS), ...Object.values(COMPARE)];
  for (const status of all) {
    assert.ok(status.label.length > 1 && TONES[status.tone], `Bad status ${JSON.stringify(status)}`);
  }
});

test('every problem code has a fix hint', () => {
  const backend = 'backend/studio_app';
  const addressCodes = [...codes(`${backend}/networking/address_plan.py`, /status="([a-z_]+)"/g),
    ...codes(`${backend}/networking/service.py`, /status="([a-z_]+)"/g)];
  const gooseProblems = codes('native/live/goose_traffic.cpp', /problem = "([a-z_]+)"/g);
  const liveStates = codes(`${backend}/acquisition/real_session.py`, /_set\("([a-z_]+)"/g);
  const compareCodes = codes(`${backend}/domain/model_compare.py`, /"(missing_on_device|extra_on_device|type_mismatch|match)"/g);
  assert.ok(addressCodes.length >= 7 && gooseProblems.length >= 4 && liveStates.length >= 5);
  for (const code of addressCodes.filter(code => code !== 'ok')) assert.ok(PROBLEMS[code]?.hint, `No hint for address status ${code}`);
  for (const code of gooseProblems) assert.ok(PROBLEMS[code]?.hint, `No hint for GOOSE problem ${code}`);
  for (const state of liveStates) assert.ok(LIVE_STATES[state]?.hint, `No hint for live state ${state}`);
  for (const code of compareCodes) assert.ok(COMPARE[code]?.hint, `No hint for comparison ${code}`);
});

test('guided components do not hard-code hints', () => {
  for (const file of guided) {
    assert.doesNotMatch(file.text, /hint:\s*['"`]/, `${file.name} hard-codes a hint; move it to src/copy`);
    assert.doesNotMatch(file.text, /<Hint\b[^>]*[^/]>/, `${file.name} passes text to <Hint>; use an id`);
  }
});
