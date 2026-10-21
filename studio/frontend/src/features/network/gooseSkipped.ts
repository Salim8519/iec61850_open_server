import type { GooseWatch } from '../../api/network';
import type { Ied } from '../../api/types';

/**
 * Words for skipped IEDs in the GOOSE traffic list. The engine leaves skipped IEDs out of its watch
 * list (they do not run), so their GOOSE would otherwise read as "not in project" although the IED
 * is in the project: this names them as skipped instead (display only; the engine is unchanged).
 */
export type SkippedGoose = { names: Set<string>; publishers: Map<string, string>; listeners: Map<string, string[]> };

export function skippedGoose(ieds: Ied[]): SkippedGoose {
  const skipped = ieds.filter(ied => ied.role === 'off');
  const publishers = new Map<string, string>();
  const listeners = new Map<string, string[]>();
  for (const ied of skipped) {
    for (const control of ied.goose?.publishes ?? []) publishers.set(control.gocb_ref, ied.name);
    for (const control of ied.goose?.subscribes ?? []) {
      listeners.set(control.gocb_ref, [...(listeners.get(control.gocb_ref) ?? []), ied.name]);
    }
  }
  return { names: new Set(skipped.map(ied => ied.name)), publishers, listeners };
}

/** The small line under the publisher's name. */
export function publisherRole(watch: GooseWatch, skipped: SkippedGoose): string {
  const role = watch.publisher?.role;
  if (role === 'real') return 'real device';
  if (role === 'outside_project') return skipped.names.has(watch.publisher?.name ?? '') ? 'skipped in this project' : 'not in project';
  return 'simulated';
}

/** The "To" cell when no running IED listens: skipped listeners are named, never "nobody". */
export function noListener(watch: GooseWatch, skipped: SkippedGoose): string {
  const names = skipped.listeners.get(watch.gocb_ref) ?? [];
  if (!names.length) return 'Nobody in project';
  return `${names.join(', ')} (skipped, so not listening)`;
}
