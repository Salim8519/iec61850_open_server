// ADR-0095: one point-kind classifier for every list; kinds come from protocol types, never names.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import {
  countKinds, dataClassOf, kindOf61850, kindOfDataClass, kindOfGatewayPoint, kindOfIec104, kindOfMasterCommand,
  kindOfMasterGroup, kindOfSnmp, KINDS,
} from '../../frontend/src/points/pointKind.ts';
import { KIND_TEXT } from '../../frontend/src/copy/pointKinds.ts';
import { kindGroup, kindMatches } from '../../frontend/src/ui/kindFilter.ts';

test('IEC 61850 attributes get their kind from FC, basic type and data class', () => {
  assert.equal(kindOf61850('ST', 'BOOLEAN'), 'SP');
  assert.equal(kindOf61850('ST', 'Dbpos'), 'DP');
  assert.equal(kindOf61850('ST', 'Enum'), 'IS');
  assert.equal(kindOf61850('ST', 'INT64'), 'IS', 'a counter needs its data class');
  assert.equal(kindOf61850('ST', 'INT64', 'BCR'), 'CT');
  assert.equal(kindOf61850('MX', 'FLOAT32'), 'MV');
  assert.equal(kindOf61850('ST', 'Quality'), 'AT');
  assert.equal(kindOf61850('MX', 'Timestamp'), 'AT');
  assert.equal(kindOf61850('CO', 'BOOLEAN'), 'CM', 'SPC and DPC both use BOOLEAN: not guessed');
  assert.equal(kindOf61850('CO', 'BOOLEAN', 'DPC'), 'DC');
  assert.equal(kindOf61850('CO', 'BOOLEAN', 'SPC'), 'SC');
  assert.equal(kindOf61850('CO', 'FLOAT32'), 'SE');
  assert.equal(kindOf61850('SP', 'FLOAT32'), 'CF');
  assert.equal(kindOf61850('DC', 'VisString255'), 'TX');
  assert.equal(kindOf61850('ST', 'Struct'), 'UN');
  assert.equal(kindOfDataClass('DPS'), 'DP');
  const classes = { 'IEDLD/XCBR1.Pos': 'DPC', 'IEDLD/XCBR1.Pos.SBOw': 'X' };
  assert.equal(dataClassOf(classes, 'IEDLD/XCBR1.Pos.Oper.ctlVal'), 'DPC');
  assert.equal(dataClassOf(classes, 'IEDLD/XCBR1.Pos'), 'DPC');
  assert.equal(dataClassOf(classes, 'IEDLD/XCBR1.PosX.stVal'), undefined, 'only whole names match');
  assert.equal(kindOfDataClass('XYZ'), 'UN');
});

test('IEC 104, gateway, master and SNMP types map onto the same kinds', () => {
  assert.equal(kindOfIec104('M_DP_TB_1'), 'DP');
  assert.equal(kindOfIec104('M_ME_NC_1'), 'MV');
  assert.equal(kindOfIec104('C_SE_NC_1'), 'SE');
  assert.equal(kindOfIec104('C_IC_NA_1'), 'UN');
  assert.equal(kindOfGatewayPoint('double'), 'DP');
  assert.equal(kindOfGatewayPoint('command_single'), 'SC');
  assert.equal(kindOfMasterGroup('double_binary'), 'DP');
  assert.equal(kindOfMasterGroup('frozen_counter'), 'CT');
  assert.equal(kindOfMasterGroup('holding_register'), 'MV');
  assert.equal(kindOfMasterCommand('setpoint_float'), 'SE');
  assert.equal(kindOfMasterCommand('registers_float'), 'SE');
  assert.equal(kindOfMasterCommand('coil'), 'SC');
  // 0178 (C6): the master's regulating step has the badge of the gateway's C_RC and command_step.
  assert.equal(kindOfMasterCommand('regulating'), kindOfIec104('C_RC_NA_1'));
  assert.equal(kindOfMasterCommand('regulating'), kindOfGatewayPoint('command_step'));
  assert.equal(kindOfSnmp('Gauge32'), 'MV');
  assert.equal(kindOfSnmp('counter64'), 'CT');
  assert.equal(kindOfSnmp('ipaddress'), 'TX');
  assert.equal(kindOfSnmp('null'), 'UN');
});

test('every kind has words and a colour token; chips and counts follow the kinds', () => {
  const tokens = readFileSync(new URL('../../frontend/src/styles/tokens.css', import.meta.url), 'utf8');
  const css = readFileSync(new URL('../../frontend/src/ui/kind.css', import.meta.url), 'utf8');
  for (const kind of KINDS) {
    assert.ok(KIND_TEXT[kind].name && KIND_TEXT[kind].glyph, kind);
    assert.match(css, new RegExp(`\\.kind-${kind}\\b`), `${kind} has a colour rule`);
  }
  for (const token of ['sp', 'dp', 'is', 'mv', 'ct', 'cmd', 'cf', 'tx']) assert.match(tokens, new RegExp(`--kind-${token}:`));
  assert.deepEqual(countKinds(['ST', 'ST', 'MX'], fc => fc === 'MX' ? 'MV' : 'SP'), { SP: 2, MV: 1 });
  const group = kindGroup();
  assert.equal(group.options.length, KINDS.length + 1);
  assert.ok(kindMatches('all', 'DP') && kindMatches('DP', 'DP') && !kindMatches('SP', 'DP'));
});

test('kind colours are readable on their tints and on white (AA)', () => {
  const tokens = readFileSync(new URL('../../frontend/src/styles/tokens.css', import.meta.url), 'utf8');
  const hex = (name: string) => tokens.match(new RegExp(`--${name}:\\s*(#[0-9a-f]{6})`, 'i'))![1];
  const lum = (h: string) => {
    const [r, g, b] = [1, 3, 5].map(i => parseInt(h.slice(i, i + 2), 16) / 255)
      .map(c => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
    return 0.2126 * r + 0.7152 * g + 0.0722 * b;
  };
  const ratio = (a: string, b: string) => { const [x, y] = [lum(a), lum(b)].sort((p, q) => q - p); return (x + 0.05) / (y + 0.05); };
  for (const token of ['sp', 'dp', 'is', 'mv', 'ct', 'cmd', 'cf', 'tx']) {
    assert.ok(ratio(hex(`kind-${token}`), hex(`kind-${token}-soft`)) >= 4.5, `${token} on its tint`);
    assert.ok(ratio(hex(`kind-${token}`), '#ffffff') >= 4.5, `${token} on white`);
  }
});
