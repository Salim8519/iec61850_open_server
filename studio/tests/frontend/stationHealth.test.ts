import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { AREAS, VERDICTS, areaInfo, verdictStatus } from '../../frontend/src/copy/health.ts';
import { TONES } from '../../frontend/src/copy/status.ts';
import { wireVerdicts } from '../../frontend/src/features/devices/signalFlow.ts';

const domain = ['goose_consistency.py', 'health_model_checks.py', 'station_health.py']
  .map(name => readFileSync(new URL(`../../backend/studio_app/domain/${name}`, import.meta.url), 'utf8')).join('\n');

test('every health result has text icon and hint', () => {
  const verdicts = new Set([...domain.matchAll(/result\("[a-z_]+", [^,]+, "([a-z_]+)"/g)].map(match => match[1]));
  for (const match of domain.matchAll(/"(ok|warning|problem|not_checked)" if /g)) verdicts.add(match[1]);
  assert.ok(verdicts.has('ok') && verdicts.has('problem') && verdicts.has('not_checked') && verdicts.has('warning'));
  for (const verdict of verdicts) {
    const status = verdictStatus(verdict);
    assert.equal(status, VERDICTS[verdict as keyof typeof VERDICTS], `verdict ${verdict} needs wording`);
    assert.ok(status.label && status.hint.length > 20 && TONES[status.tone].icon, verdict);
  }
  const labels = Object.values(VERDICTS).map(status => status.label);
  assert.equal(new Set(labels).size, labels.length, 'verdicts must read differently, not only look different');
  const areas = new Set([...domain.matchAll(/result\("([a-z_]+)"/g)].map(match => match[1]));
  for (const area of areas) assert.ok(AREAS[area] && areaInfo(area).hint.length > 20, `area ${area} needs wording`);
});

test('signal flow shows only observed wire verdicts', () => {
  const live = wireVerdicts([{ checks: [
    { area: 'goose_wire', item: 'P1LD0/LLN0$GO$g1', message: 'Arriving as engineered (5 messages).', observed_at: '2026-10-14T10:00:05+00:00' },
    { area: 'goose_wire', item: 'P1LD0/LLN0$GO$g2', message: 'The GOOSE listener is not running.', observed_at: null },
    { area: 'mode', item: 'x', message: 'On.', observed_at: '2026-10-14T10:00:05+00:00' }] }]);
  assert.deepEqual(Object.keys(live), ['P1LD0/LLN0$GO$g1']);
  assert.equal(live['P1LD0/LLN0$GO$g1'].checked_at, '10:00:05');
});
