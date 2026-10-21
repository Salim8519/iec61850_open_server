// ADR-0095: every big list can fill the window (full view), the same way everywhere.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');

test('big lists offer the shared full view with a remembered key of their own', () => {
  const panels: Record<string, string> = {
    'features/devices/DeviceDetail.tsx': 'devices', 'features/tools/gateway/GatewayTool.tsx': 'gateway',
    'features/tools/gateway/master/MasterTool.tsx': 'master', 'features/tools/gateway/converter/ConverterTool.tsx': 'converter',
    'features/tools/explorer/ExplorerTool.tsx': 'explorer', 'features/tools/snmp/read/ReadView.tsx': 'snmpRead',
    'features/tools/snmp/simulate/DevicePanel.tsx': 'snmpSimulate',
  };
  const keys = new Set<string>();
  for (const [path, key] of Object.entries(panels)) {
    const text = read(path);
    assert.match(text, new RegExp(`useFullView\\('${key}'\\)`), `${path} remembers its full view as ${key}`);
    assert.match(text, /<FullView full=\{full\}/, `${path} wraps its list in FullView`);
    assert.match(text, /<FullViewButton full=\{full\}/, `${path} shows the Full view button`);
    assert.ok(!keys.has(key), `${key} is used once`);
    keys.add(key);
  }
});

test('full view keeps the list, leaves with Esc and stays below menus', () => {
  const view = read('ui/FullView.tsx');
  assert.match(view, /event\.key !== 'Escape'/);
  assert.match(view, /closest\('input, textarea, select/, 'Esc while typing does not leave full view');
  const css = read('ui/full-view.css');
  const layer = Number(css.match(/\.full-view \{[^}]*z-index: (\d+)/)![1]);
  assert.ok(layer < 60, 'menus (70) and help bubbles (60) open above full view');
  assert.match(read('state/useFullView.ts'), /const STORE = 'full-view'/);
});
