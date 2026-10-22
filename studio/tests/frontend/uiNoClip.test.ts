// UI review 2026-10-22 (change 0160): text, buttons and icons stay whole at 1366x768; small wording fixes.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { POINTS_COPY } from '../../frontend/src/copy/snmp.ts';
import { READ_COPY } from '../../frontend/src/copy/snmpRead.ts';
import { WATCH_FILTER } from '../../frontend/src/features/tools/explorer/explorerFilters.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');
const rule = (css: string, selector: string) => {
  const start = css.indexOf(`${selector} {`);
  assert.ok(start >= 0, `missing rule ${selector}`);
  return css.slice(start, css.indexOf('}', start));
};

test('SNMP value names wrap inside their column and the import panel fits the left column', () => {
  const css = read('features/tools/snmp/snmp.css');
  assert.match(rule(css, '.gateway-points.snmp-values .editable-text'), /white-space: normal/);
  assert.match(rule(css, '.snmp-import'), /grid-template-columns: minmax\(0, 1fr\)/);
  assert.match(rule(css, '.snmp-import > button'), /white-space: normal/);
  assert.match(rule(css, '.snmp-behaviour select'), /width: 100%/);
  assert.equal(POINTS_COPY.search, 'Search name, OID or value');
  assert.equal(READ_COPY.search, 'Search name, OID, value, meaning');
});

test('gateway command rows keep both send buttons whole; the converter tabs never cut', () => {
  const css = read('features/tools/gateway/gateway.css');
  assert.match(rule(css, '.gateway-points td.value-cell.command-cell'), /overflow: visible/);
  assert.match(rule(css, '.gateway-points td.command-cell .gateway-send'), /flex-wrap: wrap/);
  assert.match(rule(css, '.converter-panel > .tabbar'), /flex-wrap: wrap/);
  assert.match(read('features/tools/gateway/PointRow.tsx'), /command \? ' command-cell' : ''/);
  assert.match(read('features/tools/gateway/converter/ConverterTool.tsx'), /gateway-panel converter-panel/);
});

test('icons keep their size and the watch list "?" is not restyled as a toolbar button', () => {
  assert.match(rule(read('styles/base.css'), 'button svg'), /flex-shrink: 0/);
  assert.match(read('features/tools/explorer/explorer.css'), /\.panel-head button:not\(\.info-tip-button\)/);
});

test('an active select keeps room for its text: the second border pixel is a shadow', () => {
  assert.doesNotMatch(read('ui/filter-bar.css'), /border-width: 2px/);
  assert.doesNotMatch(read('components/type-cell.css'), /border-width: 2px/);
  assert.match(rule(read('components/type-cell.css'), '.generator-grid .kind-select'), /width: 100%/);
});

test('the point list is not changed while locked and an empty CSV leaves the list alone', () => {
  // Save always (the Save button is gone): a locked list shows the "will save" words instead.
  assert.match(read('features/tools/gateway/PointToolbar.tsx'), /GATEWAY_COPY\.willSave : ''/);
  const table = read('features/tools/gateway/PointTable.tsx');
  assert.match(table, /if \(value\.points\.length\) edit\(value\.points, rows\)/);
  assert.match(table, /the list is unchanged/);
});

test('plain words: counters, chips and remove buttons say what they do', () => {
  assert.equal(READ_COPY.values(168, 39), '168 values · 39 kept up to date');
  const chip = WATCH_FILTER.toggles?.find(item => item.key === 'bad');
  assert.equal(chip?.label, 'Needs attention');
  assert.match(read('features/tools/explorer/WatchList.tsx'), /Remove \$\{item\.ref\} from the watch list/);
  assert.match(read('features/tools/gateway/PointRow.tsx'), /the point at address \$\{row\.address\}/);
});

test('Activity rows line up, the collapsed sidebar shows only the dot, MACs never break', () => {
  assert.doesNotMatch(rule(read('styles/workspace.css'), '.event-kind'), /display/);
  assert.match(read('features/event-log/EventLog.tsx'), /whiteSpace: 'nowrap'/);
  assert.match(read('styles/layout.css'), /\.engine-status > span:not\(\.dot\) \{ display: none; \}/);
  assert.match(read('features/network/goose.css'), /\.goose-mac \{ white-space: nowrap; \}/);
  assert.match(read('components/SignalValueEditor.tsx'), /step=\{signal\.kind\.startsWith\('INT'\) \? 1 : 'any'\}/);
});

test('Devices and SLD: sheet buttons wrap, names never go empty, copied and reasons stay honest', () => {
  assert.match(rule(read('styles/workspace.css'), '.sheet-actions'), /flex-wrap: wrap/);
  assert.match(read('features/diagram/EquipmentSheet.tsx'), /className="row sheet-actions"/);
  const name = read('features/diagram/NameField.tsx');
  assert.match(name, /if \(event\.target\.value\.trim\(\)\) change\(event\.target\.value\)/);
  assert.match(name, /onBlur=\{\(\) => \{ if \(empty\) setText\(value\); \}\}/);
  assert.match(read('features/diagram/SheetSettings.tsx'), /<NameField value=\{data\.label\}/);
  const overview = read('features/devices/DeviceOverview.tsx');
  assert.match(overview, /useEffect\(\(\) => setCopied\(false\), \[ied\.id\]\)/);
  assert.match(overview, /setTimeout\(\(\) => setCopied\(false\), 2000\)/);
  assert.match(read('features/diagram/SldImport.tsx'), /\[\.\.\.new Set\(item\.reasons\)\]\.join/);
  assert.match(read('features/devices/deviceFilters.ts'), /search: 'Search name or path'/);
});
