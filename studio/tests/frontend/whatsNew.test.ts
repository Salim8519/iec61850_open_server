import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { RELEASES, releaseOf, releasesUpTo, versionCode } from '../../frontend/src/copy/releaseNotes/index.ts';
import { noteParts, whatsNewChoice } from '../../frontend/src/features/whats-new/whatsNewChoice.ts';

const read = (path: string) => readFileSync(new URL(`../../../${path}`, import.meta.url), 'utf8');

test('release notes are newest first, one entry per version, each with words', () => {
  const codes = RELEASES.map(release => versionCode(release.version));
  assert.ok(codes.every(code => code !== null));
  assert.deepEqual([...codes].sort((a, b) => b! - a!), codes, 'newest first');
  assert.equal(new Set(codes).size, codes.length, 'one entry per version');
  for (const release of RELEASES) {
    assert.ok(release.title && release.notes.length, `${release.version} has a title and notes`);
    for (const note of release.notes) assert.doesNotMatch(note, /<|&[a-z]+;/, 'notes are plain text');
  }
});

test('version codes and lookups treat a +dev build as its release', () => {
  assert.equal(versionCode('0.1.36'), 1036);
  assert.equal(versionCode('0.1.35+dev'), 1035);
  assert.equal(versionCode('1.2.3'), 1002003);
  assert.equal(versionCode('dev'), null);
  assert.equal(releaseOf('0.1.35+dev')?.version, '0.1.35');
  assert.deepEqual(releasesUpTo('0.1.5').map(item => item.version), ['0.1.5', '0.1.4', '0.1.0']);
});

test('the website changelog is rendered from the same notes, up to the site version', () => {
  const page = read('website/src/pages/changelog.astro');
  assert.match(page, /releasesUpTo\(SITE\.version\)/);
  assert.doesNotMatch(page, /<li>[A-Z]/, 'no notes typed into the page itself');
  const site = /const VERSION = '([^']+)'/.exec(read('website/src/data/site.ts'))![1];
  assert.ok(releaseOf(site), `the site version ${site} has release notes`);
});

test("What's new opens once after an update, never on a first start", () => {
  const seen = { tour_seen_version: 1, tour_version: 1 };
  assert.equal(whatsNewChoice(seen, '0.1.36'), 'show', 'an update with no record yet');
  assert.equal(whatsNewChoice({ ...seen, whats_new_seen: 1035 }, '0.1.36'), 'show');
  assert.equal(whatsNewChoice({ ...seen, whats_new_seen: 1036 }, '0.1.36'), 'none', 'already closed');
  assert.equal(whatsNewChoice({ tour_seen_version: 0, tour_version: 1 }, '0.1.36'), 'mark', 'first start: the tour shows');
  assert.equal(whatsNewChoice(seen, '0.1.2'), 'none', 'no notes for this version');
  assert.equal(whatsNewChoice(seen, 'dev'), 'none');
});

test('notes split into their kind and words', () => {
  assert.deepEqual(noteParts('Fixed: Stop is quick'), { kind: 'Fixed', text: 'Stop is quick' });
  assert.deepEqual(noteParts('New: a pop-up'), { kind: 'New', text: 'A pop-up' });
  assert.deepEqual(noteParts('IEC 61850 IED simulation'), { kind: '', text: 'IEC 61850 IED simulation' });
});
