// Change 0093 / ADR-0064: tool pages and Devices fill the window and every list has the shared filter bar.
import assert from 'node:assert/strict';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import test from 'node:test';

const SRC = new URL('../../frontend/src/', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1');
const read = (path: string) => readFileSync(join(SRC, path), 'utf8');
const files = (folder: string): string[] => readdirSync(join(SRC, folder)).flatMap(name => {
  const path = `${folder}/${name}`;
  return statSync(join(SRC, path)).isDirectory() ? files(path) : /\.tsx?$/.test(name) ? [path] : [];
});

test('open tools and Devices use the fill-the-window page, only lists scroll', () => {
  assert.match(read('features/tools/ToolsPage.tsx'), /tool === 'traffic' \? '' : ' fill-page'/);
  assert.match(read('features/devices/DevicesPage.tsx'), /content fill-page devices-page/);
  const css = read('styles/fill.css');
  assert.match(css, /\.content\.fill-page \{ height: 100%;[^}]*overflow: hidden/);
  assert.match(css, /grid-template-rows: minmax\(0, 1fr\)/, 'the split row is the window height, not the content height');
  assert.match(css, /\.panel-body > \.table-scroll[^{]*\{\s*flex: 1 1 0; min-height: 0; max-height: none/);
  assert.match(css, /\.side-pane \{ position: relative; \}|\.side-pane \{ position: relative/);
  assert.match(css, /@media \(max-width: 1000px\), \(max-height: 620px\)/, 'narrow windows fall back to page scrolling');
  assert.match(read('main.tsx'), /import '\.\/styles\/fill\.css';/);
});

test('the SCADA tool keeps settings in a column and status with Start at its bottom', () => {
  for (const path of ['features/tools/gateway/GatewayTool.tsx', 'features/tools/gateway/master/MasterTool.tsx',
    'features/tools/gateway/converter/ConverterTool.tsx']) {
    assert.match(read(path), /fill-body fill-split scada-split/, path);
    assert.match(read(path), /<aside className="card side-pane"/, path);
    assert.match(read(path), /className="panel-body"/, path);
  }
  for (const path of ['features/tools/gateway/GatewaySettings.tsx', 'features/tools/gateway/master/MasterConnect.tsx']) {
    const source = read(path);
    assert.ok(source.indexOf('className="side-foot"') < source.indexOf('<StatusPill'), `${path}: status sits in the foot`);
    assert.match(source, /side-error/, `${path}: errors are inline text in the column`);
    assert.doesNotMatch(source, /<p className="muted"[^>]*>\{(copy|MASTER_COPY)\.intro/, `${path}: the intro moved behind "?"`);
  }
  const converter = read('features/tools/gateway/converter/ConverterTool.tsx');
  assert.match(converter, /CONVERTER_COPY\.deviceStep/);
  assert.match(converter, /CONVERTER_COPY\.scadaStep/);
  assert.match(read('features/tools/gateway/RoleSwitch.tsx'), /function SideCaption/);
  assert.match(read('features/tools/gateway/master/MasterPointTable.tsx'), /MASTER_COPY\.emptyNotConnected/);
});

test('every list uses the one FilterBar with its own remembered list name', () => {
  const lists = files('features').flatMap(path => [...read(path).matchAll(/useListFilter\('([a-z-]+)'/g)].map(match => ({ path, id: match[1] })));
  const expected = ['scada-points', 'master-points', 'traffic', 'commands', 'master-commands', 'find-results',
    'explorer-tree', 'explorer-watch', 'devices-list', 'data-points', 'live-points', 'compare', 'snmp-points',
    'snmp-read-values'];
  assert.deepEqual([...new Set(lists.map(list => list.id))].sort(), [...expected].sort());
  assert.equal(new Set(lists.map(list => list.id)).size, lists.length, 'one list name per list');
  for (const { path } of lists) assert.match(read(path), /<FilterBar /, path);
  assert.equal(files('ui').filter(path => /function FilterBar/.test(read(path))).length, 1);
});

test('filters are remembered by the backend, not by the browser', () => {
  const hook = read('state/useListFilter.ts');
  assert.match(hook, /saveListFilter\(list, stored\)/);
  for (const path of [...files('state'), ...files('ui'), ...files('features')]) {
    assert.doesNotMatch(read(path), /localStorage|sessionStorage/, `${path}: the packaged app's port can change`);
  }
  assert.match(read('api/preferences.ts'), /list_filters: \{ \[list\]: values \}/);
});

test('active chips are shown with a tick and a solid border, not colour alone', () => {
  const bar = read('ui/FilterBar.tsx');
  assert.match(bar, /\{active && <Check size=\{13\} aria-hidden \/>\}/);
  assert.match(bar, /role="radio" aria-checked=\{active\}/);
  assert.match(bar, /role="checkbox" aria-checked=\{active\}/);
  assert.match(bar, /\{filtered && <button[^>]*onClick=\{clear\}/);
  const css = read('ui/filter-bar.css');
  assert.match(css, /\.chip\.toggle \{ border: 1\.5px dashed/);
  assert.match(css, /\.chip\.toggle\.active \{ border-style: solid;/);
});

test('the Devices list row is its own component with a slot for extra hints', () => {
  const row = read('features/devices/DeviceListRow.tsx');
  assert.match(row, /hints\?: ReactNode/);
  assert.match(row, /\{hints && <span className="list-hints">\{hints\}<\/span>\}/);
  assert.match(read('features/devices/DevicesPage.tsx'), /<DeviceListRow key=\{item\.id\}/);
  assert.match(read('features/devices/DeviceDetail.tsx'), /detail-head compact/);
  assert.match(read('features/devices/DeviceOverview.tsx'), /className="stats-line"/);
});

test('the migration notice is one dismissible line with its details one click away', () => {
  const notice = read('components/CompactNotice.tsx');
  assert.match(notice, /notice compact/);
  assert.match(notice, /aria-expanded=\{open\}/);
  assert.match(notice, /notice-details/);
  const migration = read('components/MigrationNotice.tsx');
  assert.match(migration, /<CompactNotice /);
  assert.match(migration, /\{notice\.backup\}/, 'the recovery file stays visible in the details');
  assert.match(read('components/RecoveryNotice.tsx'), /problem=\{recovery\.status === 'unreadable'\}/);
  assert.match(read('styles/layout.css'), /\.notice-details \{ position: absolute;/);
});
