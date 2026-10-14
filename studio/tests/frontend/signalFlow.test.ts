import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { flowRows, LINK_STYLE, linkState, type FlowGraph } from '../../frontend/src/features/devices/signalFlow.ts';

const view = readFileSync(new URL('../../frontend/src/features/devices/SignalFlowView.tsx', import.meta.url), 'utf8');
const styles = readFileSync(new URL('../../frontend/src/features/devices/signal-flow.css', import.meta.url), 'utf8');
const edge = { id: 'e1', type: 'goose' as const, publisher: 'P1', block: 'b1', subscriber: 'S1', match: 'named' as const,
  signals: [{ source: 'P1/LD0/PTRC1.Tr.general', int_addr: 'Trip', desc: 'Trip in' }] };
const graph: FlowGraph = { source: 'x', filename: 'station.scd',
  ieds: [{ name: 'P1', desc: '', manufacturer: '', loaded: true }, { name: 'S1', desc: '', manufacturer: '', loaded: false }],
  blocks: [{ id: 'b1', type: 'goose', ied: 'P1', ref: 'P1LD0/LLN0$GO$g1', name: 'g1', dataset: 'DS1', members: [] },
    { id: 'b2', type: 'report', ied: 'P1', ref: 'P1LD0/LLN0.BR.r1', name: 'r1', dataset: 'DS2', members: [] }],
  edges: [edge], diagnostics: [], summary: { ieds: 2, goose_links: 1, report_links: 0, problems: 0, infos: 0 } };

test('renders api graph without deriving edges', () => {
  const rows = flowRows(graph, 'all', '');
  assert.deepEqual(rows.map(row => [row.block.id, row.edges.map(item => item.id)]), [['b1', ['e1']], ['b2', []]]);
  assert.equal(flowRows(graph, 'report', '').length, 1);
  assert.deepEqual(flowRows(graph, 'all', 'trip in').map(row => row.block.id), ['b1']);
  // The view only fetches the backend graph; it never resolves ExtRefs or ClientLNs itself.
  assert.match(view, /api<FlowGraph>\(`\/signal-flow\?source=/);
  assert.doesNotMatch(view, /ExtRef|ClientLN|srcCBName/);
});

test('edge types have text and line style', () => {
  assert.deepEqual(Object.keys(LINK_STYLE).sort(), ['goose', 'report']);
  const lines = new Set(Object.values(LINK_STYLE).map(style => style.line));
  assert.equal(lines.size, Object.keys(LINK_STYLE).length, 'every link type needs its own line style');
  for (const style of Object.values(LINK_STYLE)) assert.ok(style.label && style.hint.length > 20);
  assert.match(styles, /\.line-dashed[^{]*\{[^}]*border-style: dashed/);
});

test('declared edge is not shown as received', () => {
  assert.equal(linkState(edge), 'Declared in SCD');
  assert.doesNotMatch(linkState(edge), /receiv|ok|live/i);
  assert.match(view, /Receivers \(declared in SCD\)/);
});
