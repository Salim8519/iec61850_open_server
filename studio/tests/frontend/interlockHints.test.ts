import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { hint } from '../../frontend/src/copy/hints.ts';

const read = (path: string) => readFileSync(new URL(`../../frontend/src/${path}`, import.meta.url), 'utf8');

test('hints never disable switching and always name their source', () => {
  const box = read('features/diagram/SwitchHints.tsx');
  const sheet = read('features/diagram/EquipmentSheet.tsx');
  assert.doesNotMatch(box, /disabled/, 'the hint box has no way to block a command');
  const operate = sheet.slice(sheet.indexOf('<h3>Operate</h3>'), sheet.indexOf('<SwitchHints'));
  const buttons = operate.match(/<button.*?>(Open|Close)<\/button>/g) ?? [];
  assert.equal(buttons.length, 2, 'Open and Close are both in the Operate area');
  for (const button of buttons) {
    assert.match(button, /^<button (className="primary" )?disabled=\{busy\} onClick/, `${button} is only disabled while busy`);
  }
  assert.match(box, /<Hint id="switch\.hints"/);
  assert.match(hint('switch.hints'), /not from the IED/);
  assert.match(hint('switch.hints'), /never blocks/);
});
