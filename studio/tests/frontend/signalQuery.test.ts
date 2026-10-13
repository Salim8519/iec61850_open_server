import assert from 'node:assert/strict';
import test from 'node:test';
import { buildSignalQuery, signalKey } from '../../frontend/src/components/signalQuery.ts';

test('signal search defaults to all reports with a bounded page', () => {
  const query = new URLSearchParams(buildSignalQuery({ q: 'trip' }));
  assert.equal(query.get('q'), 'trip');
  assert.equal(query.get('report'), null);
  assert.equal(query.get('offset'), '0');
  assert.equal(query.get('limit'), '50');
});

test('report, kind, fc and writable filters are encoded', () => {
  const query = new URLSearchParams(buildSignalQuery({
    reportRef: 'LLN0.RP.reportCb01', offset: 50, kinds: ['BOOLEAN', 'Dbpos'],
    fc: 'ST', writableOnly: true,
  }));
  assert.equal(query.get('report'), 'LLN0.RP.reportCb01');
  assert.equal(query.get('offset'), '50');
  assert.equal(query.get('kind'), 'BOOLEAN,Dbpos');
  assert.equal(query.get('fc'), 'ST');
  assert.equal(query.get('writable'), 'true');
});

test('signal keys join IED and reference', () => {
  assert.equal(signalKey('ied-1', 'GGIO1.Ind1.stVal'), 'ied-1:GGIO1.Ind1.stVal');
});
