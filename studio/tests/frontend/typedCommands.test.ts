// ADR-0104: IED Explorer picks the way to send a value from the MMS types and the CF values it read.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { commandSpec, modelText, needsSelect, parsedNumber } from '../../frontend/src/features/tools/explorer/commandSpec.ts';
import { secondsLeft } from '../../frontend/src/features/tools/explorer/selectionTime.ts';

const leaf = (path: string, type: string, fc = path.startsWith('Oper') ? 'CO' : path === 'stVal' ? 'ST' : 'CF') =>
  ({ ref: `LD/LN.X.${path}`, fc, path, type });
const value = (raw: number) => ({ raw, text: String(raw), status: 'ok' as const, at: null });

test('each class gets its own way to send a value', () => {
  const breaker = commandSpec('Pos', [leaf('stVal', 'bit string'), leaf('Oper.ctlVal', 'boolean')], {});
  assert.equal(breaker?.kind, 'bool');
  assert.deepEqual(breaker?.choices.map(choice => choice.label), ['Close', 'Open']);
  assert.equal(commandSpec('TapChg', [leaf('valWTr.posVal', 'integer', 'ST'), leaf('Oper.ctlVal', 'bit string')], {})?.kind, 'step');
  assert.equal(commandSpec('AnOut1', [leaf('mxVal.f', 'float', 'MX'), leaf('Oper.ctlVal.f', 'float')], {})?.kind, 'float');
  assert.equal(commandSpec('ISCSO1', [leaf('stVal', 'integer'), leaf('Oper.ctlVal', 'integer')], {})?.kind, 'int');
  const mode = commandSpec('Mod', [leaf('stVal', 'integer'), leaf('Oper.ctlVal', 'integer')], {});
  assert.equal(mode?.kind, 'enum');
  assert.deepEqual(mode?.labels.map(item => item.label), ['On', 'On, blocked', 'Test', 'Test, blocked', 'Off']);
  assert.equal(commandSpec('Health', [leaf('stVal', 'integer')], {}), null, 'no Oper: nothing to send');
});

test('control model, select timeout and limits come from the values read', () => {
  const attributes = [leaf('stVal', 'integer'), leaf('Oper.ctlVal', 'integer'), leaf('ctlModel', 'integer'),
    leaf('sboTimeout', 'unsigned'), leaf('minVal', 'integer'), leaf('maxVal', 'integer')];
  const values = { 'LD/LN.X.ctlModel': value(4), 'LD/LN.X.sboTimeout': value(3000), 'LD/LN.X.minVal': value(0),
    'LD/LN.X.maxVal': value(10) };
  const spec = commandSpec('ISCSO1', attributes, values);
  assert.deepEqual([spec?.model, spec?.sboMs, spec?.min, spec?.max], [4, 3000, 0, 10]);
  assert.ok(needsSelect(4) && needsSelect(2) && !needsSelect(1) && !needsSelect(3) && !needsSelect(null));
  assert.match(modelText(4), /Select before operate/);
  assert.match(modelText(null), /not read yet/);
});

test('number boxes accept only what the kind can send', () => {
  assert.equal(parsedNumber('int', '7'), 7);
  assert.equal(parsedNumber('int', '7.5'), null);
  assert.equal(parsedNumber('float', '7.5'), 7.5);
  assert.equal(parsedNumber('float', ''), null);
  assert.equal(parsedNumber('int', 'abc'), null);
});

test('the select countdown rounds up and never goes below zero', () => {
  assert.equal(secondsLeft({ label: 'x', value: 1, until: 10_000 }, 7_100), 3);
  assert.equal(secondsLeft({ label: 'x', value: 1, until: 10_000 }, 12_000), 0);
});

test('the panel folds the command options and offers Higher/Lower and Cancel', () => {
  const panel = readFileSync(new URL('../../frontend/src/features/tools/explorer/CommandPanel.tsx', import.meta.url), 'utf8');
  const options = readFileSync(new URL('../../frontend/src/features/tools/explorer/CommandOptionsBox.tsx', import.meta.url), 'utf8');
  assert.ok(options.includes('<Disclosure title={COMMAND_COPY.optionsTitle}>'), 'options are closed by default');
  for (const word of ['COMMAND_COPY.higher', 'COMMAND_COPY.lower', 'COMMAND_COPY.cancelSelect', 'COMMAND_COPY.operate']) {
    assert.ok(panel.includes(word), word);
  }
});
