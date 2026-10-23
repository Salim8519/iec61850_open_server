// Fit to window: small windows show the big-window layout evenly smaller; big windows stay exactly as before.
import assert from 'node:assert/strict';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import test from 'node:test';
import { fitScale, layoutLimits } from '../../frontend/src/app/displayFit.ts';

const SRC = new URL('../../frontend/src/', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1');
const read = (path: string) => readFileSync(join(SRC, path), 'utf8');
const files = (folder: string, pattern: RegExp): string[] => readdirSync(join(SRC, folder)).flatMap(name => {
  const path = folder ? `${folder}/${name}` : name;
  return statSync(join(SRC, path)).isDirectory() ? files(path, pattern) : pattern.test(name) ? [path] : [];
});
const css = files('', /\.css$/);

test('fitScale: 0.05 steps, never below 0.8, 1 from about 1404×780 up', () => {
  const table: [number, number, number][] = [
    [1280, 640, 0.8], [1366, 690, 0.85], [1536, 744, 0.95], [1707, 880, 1], [1920, 1000, 1],
    [1440, 800, 1], [1404, 780, 1], [1000, 500, 0.8], [1920, 1080, 1], [2560, 1400, 1],
  ];
  for (const [width, height, fit] of table) {
    assert.equal(fitScale(width, height, 0), fit, `${width}×${height}`);
    assert.equal(fitScale(width, height, 1), 1, `${width}×${height} always 100%`);
  }
  assert.equal(fitScale(0, 0, 0), 1, 'an unmeasured window is never shrunk');
});

test('layoutLimits: the old @media thresholds on the fitted layout size', () => {
  assert.equal(layoutLimits(1280, 640, 0.8), '', '1600×800 layout: nothing narrow');
  assert.equal(layoutLimits(1280, 587, 0.8), '', '1600×734 layout: not short');
  assert.equal(layoutLimits(1100, 700, 1), 'w1599 w1440 w1100');
  assert.equal(layoutLimits(1920, 1000, 1), '');
  assert.equal(layoutLimits(1440, 900, 1), 'w1599 w1440');
  assert.match(layoutLimits(1280, 600, fitScale(1280, 600, 1)), /\bh620\b/, 'always 100% keeps today\'s short-window rule');
  assert.equal(layoutLimits(1000, 500, 0.8), 'w1599 w1440', '1250×625 layout');
  assert.equal(layoutLimits(1000, 480, 0.8), 'w1599 w1440 h620', '1250×600 layout');
  assert.equal(layoutLimits(0, 0, 1), '', 'an unmeasured (hidden) window gets no narrow rules');
  assert.equal(layoutLimits(500, 400, 1), 'w1599 w1440 w1100 w1000 w520 h620');
});

test('every vh/vw in the CSS is divided by the fit, so the fitted page still fills the window', () => {
  assert.ok(css.length > 20, 'the stylesheets were found');
  for (const path of css) {
    for (const line of read(path).split('\n')) {
      for (const use of line.match(/\d+v[hw]\b[^;]*/g) ?? []) {
        assert.match(use, /^\d+v[hw] \/ var\(--fit, 1\)/, `${path}: ${line.trim()}`);
      }
    }
  }
  assert.match(read('styles/layout.css'), /\.app \{ height: calc\(100vh \/ var\(--fit, 1\)\);/);
  assert.match(read('styles/tokens.css'), /:root \{ zoom: var\(--fit, 1\); \}/);
});

test('no size @media is left: narrow-window rules use the data-max tokens from installFit', () => {
  for (const path of css) assert.doesNotMatch(read(path), /@media[^{]*\((max|min)-(width|height)/, path);
});

test('the SLD canvas cancels the page zoom and its panels scale back', () => {
  const canvas = read('styles/canvas-interaction.css');
  assert.match(canvas, /\.react-flow \{ zoom: calc\(1 \/ var\(--fit, 1\)\); \}/);
  assert.match(canvas, /\.react-flow__panel, \.stray-notice \{ zoom: var\(--fit, 1\); \}/);
});

test('main.tsx installs fit before the first render', () => {
  const main = read('main.tsx');
  assert.match(main, /import \{ installFit \} from '\.\/app\/installFit';/);
  assert.ok(main.indexOf('installFit();') > 0 && main.indexOf('installFit();') < main.indexOf('createRoot('));
});

test('popups placed by pixels convert into their zoom space; no raw window size', () => {
  for (const path of ['ui/Menu.tsx', 'ui/HelpTip.tsx', 'ui/VirtualRows.tsx', 'features/diagram/TestMenu.tsx',
    'features/tour/TourOverlay.tsx']) {
    const source = read(path);
    assert.match(source, /from '[./]+(ui\/)?zoomSpace'/, path);
    assert.doesNotMatch(source, /\binner(Width|Height)\b/, path);
  }
});
