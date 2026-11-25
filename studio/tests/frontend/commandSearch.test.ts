// ADR-0149: the Commands pop-up's search and the Data points rows that can send a command.
import assert from 'node:assert/strict';
import test from 'node:test';
import { commandRefOf, matchesSearch } from '../../frontend/src/features/devices/commandSearch.ts';

const position = { ref: 'E10_BCUSystem/DiscCSWI3.Pos', cdc: 'DPC' };

test('every typed word must be in the reference or the class, any case', () => {
  assert.equal(matchesSearch(position, ''), true);
  assert.equal(matchesSearch(position, 'cswi3'), true);
  assert.equal(matchesSearch(position, 'CSWI3 dpc'), true);
  assert.equal(matchesSearch(position, 'CSWI4'), false);
  assert.equal(matchesSearch(position, 'disc spc'), false);
});

test('only the ctlVal row of Oper belongs to a command', () => {
  assert.equal(commandRefOf('E10_BCUSystem/DiscCSWI3.Pos.Oper.ctlVal'), 'E10_BCUSystem/DiscCSWI3.Pos');
  assert.equal(commandRefOf('E10_BCUMaster/GGIO1.AnOut1.Oper.ctlVal.f'), 'E10_BCUMaster/GGIO1.AnOut1');
  assert.equal(commandRefOf('E10_BCUSystem/DiscCSWI3.Pos.Oper.origin.orCat'), null);
  assert.equal(commandRefOf('E10_BCUSystem/DiscCSWI3.Pos.stVal'), null);
  assert.equal(commandRefOf('E10_BCUSystem/DiscCSWI3.Pos.SBOw.ctlVal'), null);
});
