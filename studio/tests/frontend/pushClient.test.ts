import assert from 'node:assert/strict';
import test from 'node:test';
import { fetchCapabilities, pathTopic, readCapabilities, type Capabilities } from '../../frontend/src/api/capabilities.ts';
import { createLiveFeed } from '../../frontend/src/api/liveFeed.ts';
import { createPushClient } from '../../frontend/src/api/pushClient.ts';
import { fakeTimers, fakeVisibility, settle } from './fakeClock.ts';
import { FakeSource, fullFrame, queue, topicsEndpoint } from './pushFakes.ts';

const CAPS: Capabilities = { epoch: 'e1', push: 1, topics: ['activity', 'gateway.points', 'gateway.traffic', 'health'],
  paths: { '/health': { t: 'health', params: [] }, '/activity': { t: 'activity', params: [] },
    '/tools/gateway/points': { t: 'gateway.points', params: ['protocol'] },
    '/tools/gateway/traffic': { t: 'gateway.traffic', params: [] } } };

test('capabilities: only push protocol 1 counts; Python\'s 404 and errors keep polling', async () => {
  assert.equal(readCapabilities(JSON.stringify(CAPS))?.push, 1);
  assert.equal(readCapabilities(JSON.stringify({ ...CAPS, push: 2 })), null);
  assert.equal(readCapabilities('{"error":"Not found"}'), null);
  assert.equal(readCapabilities('<html>'), null);
  const asked: { input: string; init?: RequestInit }[] = [];
  const answer = (ok: boolean, text: string) => async (input: string, init?: RequestInit) => {
    asked.push({ input, init }); return { ok, text: async () => text }; };
  assert.equal(await fetchCapabilities(answer(false, '{"error":"Not found"}'), 'http://127.0.0.1:1'), null);
  assert.deepEqual(asked[0], { input: '/api/v1/capabilities',
    init: { credentials: 'same-origin', headers: { 'X-Studio-Frontend': 'http://127.0.0.1:1' } } });
  assert.equal(await fetchCapabilities(async () => { throw new Error('offline'); }, ''), null);
  assert.equal((await fetchCapabilities(answer(true, JSON.stringify(CAPS)), ''))?.epoch, 'e1');
});

test('capabilities: a path is covered when its query names only the topic\'s parameters', () => {
  assert.deepEqual(pathTopic(CAPS.paths, '/tools/gateway/points?protocol=iec104'),
    { t: 'gateway.points', params: { protocol: 'iec104' }, key: '[["protocol","iec104"]]' });
  assert.equal(pathTopic(CAPS.paths, '/health')?.t, 'health');
  assert.equal(pathTopic(CAPS.paths, '/tools/gateway/traffic?after=12'), null, 'traffic pages stay polled');
  assert.equal(pathTopic(CAPS.paths, '/health?x=1'), null);
  assert.equal(pathTopic(CAPS.paths, '/tools/gateway/points?protocol=a&protocol=b'), null);
  assert.equal(pathTopic(CAPS.paths, '/network/adapters?refresh=1'), null);
  assert.equal(pathTopic(CAPS.paths, '/constructor'), null);
});

function rig(caps: Capabilities | null = CAPS) {
  const sources: FakeSource[] = [];
  const endpoint = topicsEndpoint();
  const soon = queue();
  const later = queue();
  const view = fakeVisibility();
  const clock = fakeTimers();
  const answers: Record<string, string> = { '/health': '{"ok":1}\n', '/activity': '{"items":[]}\n',
    '/tools/gateway/points?protocol=iec104': '{"points":[]}\n', '/tools/gateway/points?protocol=iec61850': '{"points":[]}\n',
    '/tools/gateway/traffic?after=0': '{"entries":[]}\n' };
  const asked: string[] = [];
  const feed = createLiveFeed(async path => { asked.push(path); return answers[path]; }, { timers: clock.timers });
  const plain: string[] = [];
  let capsAsked = 0;
  const client = createPushClient({
    capabilities: async () => { capsAsked += 1; return caps; },
    open: url => { const source = new FakeSource(url); sources.push(source); return source; },
    put: endpoint.put, tab: () => 'tab-1', visibility: view.visibility, soon: soon.add, later: later.add,
    feed, plain: () => { plain.push('events'); return { close: () => plain.push('closed') }; },
  });
  const bus: unknown[] = [];
  let opens = 0;
  const stream = client.factory({ open: () => { opens += 1; }, event: event => bus.push(event) });
  return { sources, endpoint, soon, later, view, clock, feed, client, stream, bus, plain, asked, answers,
    opens: () => opens, capsAsked: () => capsAsked, last: () => sources[sources.length - 1] };
}

test('push client: Python (no capabilities) keeps the plain stream and every poll', async () => {
  const r = rig(null);
  r.feed.subscribe('/health', 3000, () => {});
  await settle();
  assert.deepEqual(r.plain, ['events'], 'today\'s /api/v1/events stream');
  assert.equal(r.sources.length, 0, 'no push stream');
  assert.deepEqual(r.clock.intervals(), [3000], 'the poll timer stays');
  assert.deepEqual(r.feed.pushed(), []);
  assert.equal(r.endpoint.bodies.length, 0);
  r.stream.close();
  assert.deepEqual(r.plain, ['events', 'closed']);
});

test('push client: bus events pass through; frames replace the covered polls; one PUT per frame', async () => {
  const r = rig();
  const health: unknown[] = [];
  const traffic: unknown[] = [];
  r.feed.subscribe('/health', 3000, value => health.push(value));
  r.feed.subscribe('/tools/gateway/traffic?after=0', 2000, value => traffic.push(value));
  await settle();
  const source = r.last();
  assert.equal(source.url, '/api/v1/push?client=tab-1&after=0');
  source.open();
  assert.equal(r.opens(), 1);
  source.bus(7, { id: 7, kind: 'signal', timestamp: 1 });
  assert.deepEqual(r.bus, [{ id: 7, kind: 'signal', timestamp: 1 }]);
  source.hello('e1');
  assert.deepEqual(r.feed.pushed(), ['/health'], 'traffic ?after= pages stay polled');
  assert.deepEqual(r.clock.intervals(), [2000]);
  r.feed.subscribe('/activity', 2000, () => {});
  r.soon.run();
  assert.deepEqual(r.endpoint.bodies, [{ client: 'tab-1', topics: [{ t: 'health', params: {} }, { t: 'activity', params: {} }],
    resync: [] }], 'one PUT for every change made before the frame');
  source.frame(fullFrame('health', 1, '{"ok":1}'));
  assert.equal(health.length, 1, 'the same text as the poll answer is not shown again');
  source.frame(fullFrame('health', 2, '{"ok":2}'));
  assert.deepEqual(health, [{ ok: 1 }, { ok: 2 }]);
  const stop = r.feed.subscribe('/tools/gateway/points?protocol=iec104', 1000, () => {});
  r.soon.run();
  assert.equal(r.endpoint.bodies.length, 1, 'no second PUT while one is answered');
  r.endpoint.answer();
  await settle();
  r.soon.run();
  assert.deepEqual(r.endpoint.bodies[1].topics.map(topic => topic.t), ['health', 'activity', 'gateway.points']);
  assert.deepEqual(r.endpoint.bodies[1].topics[2].params, { protocol: 'iec104' });
  stop();
  r.endpoint.answer();
  await settle();
  r.soon.run();
  assert.deepEqual(r.endpoint.bodies[2].topics.map(topic => topic.t), ['health', 'activity']);
});

test('push client: hidden sends [], visible sends the list again', async () => {
  const r = rig();
  r.feed.subscribe('/health', 3000, () => {});
  await settle();
  r.last().hello('e1');
  r.soon.run(); r.endpoint.answer(); await settle();
  r.view.set(true);
  r.soon.run();
  assert.deepEqual(r.endpoint.bodies[1].topics, [], 'a hidden tab asks the server for nothing');
  r.endpoint.answer(); await settle();
  r.view.set(false);
  r.soon.run();
  assert.deepEqual(r.endpoint.bodies[2].topics, [{ t: 'health', params: {} }]);
});

test('push client: a delta it cannot apply asks for resync; a status frame asks over HTTP', async () => {
  const r = rig();
  const seen: unknown[] = [];
  const errors: string[] = [];
  r.feed.subscribe('/activity', 2000, value => seen.push(value), error => errors.push(error.message));
  await settle();
  r.last().hello('e1');
  r.soon.run(); r.endpoint.answer(); await settle();
  r.last().frame(fullFrame('activity', 3, '{"items":[]}'));
  r.last().frame('{"t":"activity","rev":5,"base":4,"upsert":{"a":"{}"}}');
  r.soon.run();
  assert.deepEqual(r.endpoint.bodies[1].resync, ['activity']);
  const asked = r.asked.length;
  r.last().frame(fullFrame('activity', 6, '{"error":"Explorer stopped"}', 500));
  await settle();
  assert.equal(r.asked.length, asked + 1, 'the error comes from a real GET, exactly as a poll would show it');
});

test('push client: the last path of a topic wins; the other goes back to polling', async () => {
  const r = rig();
  await settle();
  r.last().hello('e1');
  const a: unknown[] = [];
  const b: unknown[] = [];
  r.feed.subscribe('/tools/gateway/points?protocol=iec104', 1000, value => a.push(value));
  r.feed.subscribe('/tools/gateway/points?protocol=iec61850', 1000, value => b.push(value));
  assert.deepEqual(r.feed.pushed(), ['/tools/gateway/points?protocol=iec61850']);
  assert.deepEqual(r.clock.intervals(), [1000], 'the earlier path polls again');
  r.soon.run();
  assert.deepEqual(r.endpoint.bodies[0].topics, [{ t: 'gateway.points', params: { protocol: 'iec61850' } }]);
});

test('push client: changed parameters drop frames of the old ones until the PUT is answered', async () => {
  const r = rig();
  const seen: unknown[] = [];
  let stop = r.feed.subscribe('/tools/gateway/points?protocol=iec104', 1000, () => {});
  await settle();
  r.last().hello('e1');
  r.soon.run(); r.endpoint.answer(); await settle();
  stop();
  stop = r.feed.subscribe('/tools/gateway/points?protocol=iec61850', 1000, value => seen.push(value));
  await settle();
  const body = { head: '{"points":[', keys: ['x'], rows: new Map([['x', '{"id":"x"}']]), tail: ']}' };
  r.last().frame(fullFrame('gateway.points', 9, body)); // still iec104, already on the wire
  assert.deepEqual(seen, [{ points: [] }], 'only the HTTP answer of the new path');
  r.soon.run(); r.endpoint.answer(); await settle();
  r.soon.run();
  assert.deepEqual(r.endpoint.bodies[2].resync, ['gateway.points'], 'a dropped full frame might have been the new one');
  r.last().frame(fullFrame('gateway.points', 1, body));
  assert.deepEqual(seen, [{ points: [] }, { points: [{ id: 'x' }] }]);
});

test('push client: stream failure polls; reconnect pushes again; a new epoch resubscribes', async () => {
  const r = rig();
  r.feed.subscribe('/health', 3000, () => {});
  await settle();
  const first = r.last();
  first.hello('e1');
  first.bus(41, { id: 41, kind: 'runtime', timestamp: 1 });
  r.soon.run(); r.endpoint.answer(); await settle();
  first.fail(false);
  assert.deepEqual(r.clock.intervals(), [3000], 'polling while the stream reconnects by itself');
  first.hello('e2');
  assert.deepEqual(r.clock.intervals(), []);
  r.soon.run();
  assert.equal(r.endpoint.bodies.length, 2, 'the restarted server gets the list again');
  r.endpoint.answer(); await settle();
  first.fail(true);
  assert.equal(r.later.size(), 1, 'a closed stream is retried later');
  assert.deepEqual(r.clock.intervals(), [3000]);
  r.later.run(); await settle();
  assert.equal(r.capsAsked(), 2, 'capabilities are asked again before reconnecting');
  assert.equal(r.last().url, '/api/v1/push?client=tab-1&after=41', 'resumes after the last bus event');
  r.last().hello('e2');
  assert.deepEqual(r.clock.intervals(), []);
  r.stream.close();
  assert.ok(r.last().closed);
  assert.deepEqual(r.clock.intervals(), [3000], 'closing the stream gives the views back to polling');
});
