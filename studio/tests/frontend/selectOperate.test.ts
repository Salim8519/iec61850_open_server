import assert from 'node:assert/strict';
import test from 'node:test';
import { isSbo, modelName, SBO_IDLE, sboBusy, sboReducer, secondsLeft, type SboEvent, type SboState }
  from '../../frontend/src/equipment/selectOperate.ts';
import { changedCount, commandBody, DEFAULT_OPTIONS, optionsToSend, startOptions }
  from '../../frontend/src/equipment/commandOptions.ts';
import { commandInput, commandWords, controlSummary, enumChoices, parseCommandNumber, rangeHint }
  from '../../frontend/src/equipment/typedCommand.ts';
import { cancelBody, commandChoices, commandControls, controlOf, selectBody, sendBody }
  from '../../frontend/src/equipment/pointRequests.ts';
import { kindOfPoint } from '../../frontend/src/equipment/pointDisplay.ts';
import type { ExtendedControl, Ied, PanelPoint } from '../../frontend/src/api/types.ts';

const run = (events: SboEvent<string>[], start: SboState<string> = SBO_IDLE) =>
  events.reduce((state: SboState<string>, event) => sboReducer(state, event), start);

test('models: the numeric ctlModel and the typed words both say whether Select comes first (ADR-0108)', () => {
  assert.deepEqual([0, 1, 2, 3, 4, 5].map(modelName), ['status-only', 'direct-normal', 'sbo-normal', 'direct-enhanced', 'sbo-enhanced', '']);
  assert.equal(modelName('sbo-enhanced'), 'sbo-enhanced');
  assert.equal(modelName('select'), '');
  assert.deepEqual([4, 2, 'sbo-normal', 3, 1, 'direct-enhanced', undefined, null].map(isSbo),
    [true, true, true, false, false, false, false, false]);
});

test('select, countdown and operate: the happy path', () => {
  const selected = run([{ type: 'select', choice: 'close' }, { type: 'selected', windowMs: 27000, now: 1000 }]);
  assert.equal(selected.phase, 'selected');
  assert.equal(selected.choice, 'close');
  assert.equal(selected.deadline, 28000);
  assert.equal(secondsLeft(selected, 1000), 27);
  assert.equal(secondsLeft(selected, 1001), 27, 'whole seconds round up so 0 s is never shown while it still holds');
  assert.equal(secondsLeft(selected, 27500), 1);
  assert.ok(sboBusy(selected));
  const done = run([{ type: 'operate' }, { type: 'operated', message: 'Close done' }], selected);
  assert.deepEqual([done.phase, done.tone, done.message, done.deadline], ['idle', 'done', 'Close done', null]);
  assert.ok(!sboBusy(done));
});

test('the selection runs out at its deadline, and late answers change nothing', () => {
  const selected = run([{ type: 'select', choice: 'open' }, { type: 'selected', windowMs: 7000, now: 0 }]);
  assert.equal(run([{ type: 'tick', now: 6999 }], selected).phase, 'selected');
  const expired = run([{ type: 'tick', now: 7000 }], selected);
  assert.deepEqual([expired.phase, expired.tone], ['idle', 'expired']);
  assert.equal(secondsLeft(expired, 7000), 0);
  assert.equal(run([{ type: 'operate' }], expired).phase, 'idle', 'Operate after the deadline does nothing');
  assert.equal(run([{ type: 'operated', message: 'late' }], expired).tone, 'expired');
  assert.equal(run([{ type: 'selected', windowMs: 9000, now: 0 }], expired).phase, 'idle', 'a late select answer revives nothing');
  assert.equal(run([{ type: 'clear' }], expired).tone, 'none');
});

test('cancel, failures and a second select while busy', () => {
  const selected = run([{ type: 'select', choice: 'close' }, { type: 'selected', windowMs: 27000, now: 0 }]);
  const cancelled = run([{ type: 'cancel' }, { type: 'cancelled', message: '' }], selected);
  assert.deepEqual([cancelled.phase, cancelled.tone], ['idle', 'cancelled']);
  const refused = run([{ type: 'select', choice: 'close' }, { type: 'failed', message: 'Real device' }]);
  assert.deepEqual([refused.phase, refused.tone, refused.message], ['idle', 'failed', 'Real device']);
  assert.equal(run([{ type: 'select', choice: 'open' }], selected).choice, 'close', 'one selection at a time');
  assert.equal(run([{ type: 'selected', windowMs: 0, now: 0 }], run([{ type: 'select', choice: 'x' }])).tone, 'expired');
  const failedOperate = run([{ type: 'operate' }, { type: 'failed', message: 'ran out' }], selected);
  assert.deepEqual([failedOperate.phase, failedOperate.tone], ['idle', 'failed']);
});

test('command options: nothing is sent until they are used, so direct requests stay byte-identical', () => {
  assert.deepEqual(commandBody('close', optionsToSend(undefined, DEFAULT_OPTIONS)), { action: 'close' });
  assert.equal(JSON.stringify(commandBody('open', undefined)), '{"action":"open"}');
  const test = { ...DEFAULT_OPTIONS, test: true, orcat: 2 };
  assert.deepEqual(commandBody('close', optionsToSend(undefined, test)), { action: 'close', options: test });
  assert.deepEqual(optionsToSend(test, DEFAULT_OPTIONS), DEFAULT_OPTIONS, 'going back to the defaults is remembered too');
  assert.deepEqual(startOptions({ ...DEFAULT_OPTIONS, orident: 'bench' }).orident, 'bench');
  assert.equal(changedCount(test), 2);
});

// Typed controls exactly as GET /controls/typed answers them (ADR-0104 `extended_controls`).
const inc: ExtendedControl = { ref: 'LD0/ISCSO1', status: 'LD0/ISCSO1.stVal', cdc: 'INC', ctl_val_kind: 'INT32',
  ctl_val_ref: 'LD0/ISCSO1.Oper.ctlVal', value_kind: 'int', model: 2, sbo_timeout_ms: 30000, min: 0, max: 10, step: 1,
  labels: {}, trans_ind: null };
const enc: ExtendedControl = { ...inc, ref: 'LD0/LLN0.Mod', cdc: 'ENC', ctl_val_kind: 'Enum', value_kind: 'enum', model: 1,
  min: null, max: null, step: null, labels: { on: 1, blocked: 2, test: 3 } };
const apc: ExtendedControl = { ...inc, ref: 'LD0/APCSO1', cdc: 'APC', value_kind: 'float', min: -5, max: 5, step: null };
const bsc: ExtendedControl = { ...inc, ref: 'LD0/TCTR1.TapChg', cdc: 'BSC', value_kind: 'step', min: -64, max: 63, step: 1,
  trans_ind: 'LD0/TCTR1.TapChg.valWTr.transInd' };
const bac: ExtendedControl = { ...bsc, ref: 'LD0/BAC1', cdc: 'BAC' };
const row = (unit = ''): PanelPoint => ({ id: 'p1', label: 'Tap', group: '', type: 'command', source: null,
  control: { ied_id: 'ied1', ref: inc.ref }, unit, scale: 1, offset: 0, decimals: 2, pinned: false, healthy: null });

test('typed command rows pick their input, words and kind badge', () => {
  assert.deepEqual([inc, enc, apc, bsc, bac, { cdc: 'SPC' as const }, undefined].map(commandInput),
    ['number', 'list', 'number', 'steps', 'steps', 'choices', 'none']);
  assert.deepEqual([inc.model, enc.model].map(isSbo), [true, false], 'typed models are the numeric ctlModel');
  assert.equal(rangeHint(inc), 'from 0 to 10, steps of 1');
  assert.equal(rangeHint(enc), '');
  assert.deepEqual(parseCommandNumber(inc, '4'), { value: 4 });
  assert.match(parseCommandNumber(inc, '4.5').error ?? '', /whole numbers/);
  assert.deepEqual(parseCommandNumber(apc, '2,5'), { value: 2.5 });
  assert.match(parseCommandNumber(apc, 'x').error ?? '', /Type a number/);
  assert.deepEqual(enumChoices(enc), ['on', 'blocked', 'test']);
  assert.equal(commandWords(row('kV'), apc, 2.5), '2.5 kV');
  assert.equal(commandWords(row('kV'), inc, 3), '3');
  assert.deepEqual(['higher', 'lower'].map(value => commandWords(row(), bsc, value)), ['Raise', 'Lower']);
  assert.equal(commandWords(row(), { ...inc, cdc: 'DPC' }, true), 'Close');
  assert.equal(controlSummary(inc), 'whole number from 0 to 10, steps of 1');
  assert.equal(controlSummary(enc), 'list: on, blocked, test');
  assert.equal(controlSummary(bsc), 'Lower / Raise');
  assert.deepEqual([inc, enc, apc, bsc, bac].map(control => kindOfPoint(row(), control)), ['SE', 'SE', 'SE', 'SE', 'SE']);
  assert.deepEqual(commandChoices(inc), [], 'typed classes have their own inputs');
});

test('panel requests: select, cancel and typed sends use the existing test route', () => {
  assert.deepEqual(selectBody(row(), 4), { action: 'select', row: 'p1', value: 4 });
  assert.deepEqual(cancelBody(row()), { action: 'cancel', row: 'p1' });
  assert.deepEqual(sendBody(row(), 'blocked'), { action: 'send', row: 'p1', value: 'blocked' });
  const ied = { controls: [{ ref: 'LD0/CSWI1.Pos', status: 's', cdc: 'DPC', ctl_val_kind: 'BOOLEAN', model: 4, sbo_timeout_ms: 30000,
    delay_ms: 0, blocked: false }] } as unknown as Ied;
  const typed = [inc, { ...enc, ref: 'LD0/CSWI1.Pos' }, bac, { ...inc, ref: 'LD0/X', cdc: 'LTRK' } as unknown as ExtendedControl];
  assert.deepEqual(commandControls(ied, typed).map(control => control.ref), ['LD0/CSWI1.Pos', 'LD0/ISCSO1', 'LD0/BAC1'],
    'controls first; a typed entry of the same ref and classes nobody can send are left out');
  assert.equal(controlOf(row(), ied, typed)?.cdc, 'INC');
  assert.equal(controlOf(row(), ied, []), undefined, 'engines before ADR-0104 send no typed list');
});
