// Fix plan A15: an error while drawing shows a message and Reload, never a blank window.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { ErrorBoundary } from '../../frontend/src/app/ErrorBoundary.ts';
import { CRASH_COPY } from '../../frontend/src/copy/crash.ts';

/** Every piece of text in a React element tree. */
function texts(node: unknown): string[] {
  if (typeof node === 'string') return [node];
  if (Array.isArray(node)) return node.flatMap(texts);
  if (node && typeof node === 'object' && 'props' in node) {
    return texts((node as { props: { children?: unknown } }).props.children);
  }
  return [];
}

test('a child that throws makes the boundary show the message and Reload', () => {
  const child = { type: 'div', props: { children: 'the app' } };
  const boundary = new ErrorBoundary({ children: child as never });
  assert.equal(boundary.render(), child, 'no error: the app itself is drawn');
  boundary.state = { ...boundary.state, ...ErrorBoundary.getDerivedStateFromError() };
  const shown = texts(boundary.render());
  assert.deepEqual(shown, [CRASH_COPY.title, CRASH_COPY.detail, CRASH_COPY.reload]);
  assert.equal(CRASH_COPY.title, 'Something went wrong. Your IEDs are still running.');
});

test('the whole window, licence screen included, is inside the boundary', () => {
  const main = readFileSync(new URL('../../frontend/src/main.tsx', import.meta.url), 'utf8');
  assert.match(main, /<ErrorBoundary><LicenseBoundary><App \/><\/LicenseBoundary><\/ErrorBoundary>/);
});
