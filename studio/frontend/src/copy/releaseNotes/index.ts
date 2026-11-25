/** The one list of release notes (change 0238). The app shows it in What's new and the website's
 *  changelog page renders the same list, so the two never disagree. Add the next version at the top
 *  of the newest file; the website shows it only once its own version reaches it. */
import { RELEASES_000 } from './notes000.ts';
import { RELEASES_020 } from './notes020.ts';
import { RELEASES_030 } from './notes030.ts';
import type { Release } from './types.ts';

export type { Release } from './types.ts';

export const RELEASES: Release[] = [...RELEASES_030, ...RELEASES_020, ...RELEASES_000];

/** "0.1.36" or "0.1.36+dev" as one whole number (1036), or null when it is not a version. */
export function versionCode(version: string): number | null {
  const match = /^(\d+)\.(\d+)\.(\d+)(?:[+-].*)?$/.exec(version.trim());
  if (!match) return null;
  const [major, minor, patch] = match.slice(1).map(Number);
  return minor > 999 || patch > 999 ? null : major * 1_000_000 + minor * 1000 + patch;
}

/** The notes for versions up to and including `version`, newest first. */
export function releasesUpTo(version: string): Release[] {
  const limit = versionCode(version);
  return limit === null ? [] : RELEASES.filter(release => (versionCode(release.version) ?? Infinity) <= limit);
}

/** The notes of exactly this version (a "+dev" build counts as its release), if there are any. */
export function releaseOf(version: string): Release | undefined {
  const code = versionCode(version);
  return code === null ? undefined : RELEASES.find(release => versionCode(release.version) === code);
}
