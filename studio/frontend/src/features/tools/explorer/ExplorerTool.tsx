import { Loader2, Plug, Telescope, Unplug } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { EXPLORER_STATUS, explorerApi, type Attribute, type ExplorerStatus } from '../../../api/tools';
import { liveFeed } from '../../../api/polls';
import { EXPLORER_COPY } from '../../../copy/tools';
import { HelpTip } from '../../../ui/HelpTip';
import { InfoTip } from '../../../ui/InfoTip';
import { StatusPill } from '../../../ui/StatusPill';
import { TabBar } from '../../../ui/TabBar';
import type { Status } from '../../../copy/status';
import { ModelTree } from './ModelTree';
import { NodeDetail } from './NodeDetail';
import { WatchList } from './WatchList';
import './explorer.css';

const PILL: Record<ExplorerStatus['state'], Status> = {
  disconnected: { label: 'Not connected', tone: 'off', hint: 'Type an address and press Connect.' },
  connecting: { label: 'Connecting', tone: 'waiting', hint: 'Opening an IEC 61850 association.' },
  connected: { label: EXPLORER_COPY.readOnly, tone: 'good', hint: 'Explorer reads this device and sends commands only after you confirm them.' },
  failed: { label: 'Could not connect', tone: 'problem', hint: 'See the message below.' },
  lost: { label: 'Connection lost', tone: 'problem', hint: 'The device closed the connection. Press Connect to try again.' },
};

type View = 'node' | 'watch';

/** IED Explorer: connect by address, browse one level at a time, and watch chosen values. A toolbar row,
 *  then the model tree on the left and one full-height panel on the right whose tabs switch between
 *  the chosen node and the watch list, so small screens get the whole height for either. */
export function ExplorerTool({ target }: { target?: { host: string; port: number } }) {
  const [status, setStatus] = useState<ExplorerStatus | null>(null);
  const [host, setHost] = useState(target?.host ?? '');
  const [port, setPort] = useState(String(target?.port ?? 102));
  const [node, setNode] = useState('');
  const [view, setView] = useState<View>('node');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const run = useCallback((task: () => Promise<ExplorerStatus>) => {
    setBusy(true); setError('');
    task().then(setStatus).catch(reason => setError(reason.message)).finally(() => setBusy(false));
  }, []);
  const connect = useCallback((address: string, number: number) => {
    setNode('');
    run(() => explorerApi.connect(address.trim(), number));
  }, [run]);
  // Open each "Explore" target once: React StrictMode runs effects twice in development, and two
  // overlapping connects to the same device used to leave the live worker answering "Already connected".
  const opened = useRef<typeof target>(undefined);
  useEffect(() => {
    const fresh = opened.current !== target;
    opened.current = target;
    explorerApi.status().then(value => {
      setStatus(value);
      if (target && fresh && !(value.state === 'connected' && value.host === target.host && value.port === target.port))
        connect(target.host, target.port);
      else if (value.host && !target) { setHost(value.host); setPort(String(value.port)); }
    }).catch(reason => setError(reason.message));
  }, [target, connect]);
  const connected = status?.state === 'connected';
  // Shared live feed (speed lane F): same intervals as before, no overlap, unchanged answers ignored.
  const every = status?.polling ? Math.max(500, Math.min(status.interval_ms, 2000)) : 4000;
  useEffect(() => {
    if (!connected) return undefined;
    return liveFeed.subscribe<ExplorerStatus>(EXPLORER_STATUS, every, setStatus);
  }, [connected, every]);
  const watched = new Set((status?.watch ?? []).map(item => `${item.ref}|${item.fc}`));
  const addWatch = (items: Attribute[]) => status && run(() =>
    explorerApi.watch([...status.watch, ...items.map(({ ref, fc, type }) => ({ ref, fc, type }))]));
  const pending = busy && status?.state !== 'connected';
  return <>
    <form className="card explorer-bar" onSubmit={event => { event.preventDefault(); connect(host, Number(port)); }}>
      <label className="explorer-host">{EXPLORER_COPY.address}
        <input value={host} onChange={event => setHost(event.target.value)} placeholder="192.168.1.10"
          inputMode="decimal" autoComplete="off" spellCheck={false} disabled={connected} /></label>
      <label className="explorer-port"><InfoTip term="mms" label={EXPLORER_COPY.port} />
        <input value={port} onChange={event => setPort(event.target.value.replace(/\D/g, ''))} inputMode="numeric"
          disabled={connected} /></label>
      {connected
        ? <button type="button" onClick={() => { setNode(''); run(explorerApi.disconnect); }} disabled={busy}>
          <Unplug size={16} />{EXPLORER_COPY.disconnect}</button>
        : <button type="submit" className="primary" disabled={pending || !host.trim()}>
          {pending ? <Loader2 size={16} className="spin" /> : <Plug size={16} />}{EXPLORER_COPY.connect}</button>}
      {status && <StatusPill status={pending ? PILL.connecting : PILL[status.state]} />}
      <HelpTip text={EXPLORER_COPY.intro} label="How does IED Explorer work?" />
      {(error || ((status?.state === 'lost' || status?.state === 'failed') && status.message)) &&
        <p className="explorer-error explorer-bar-error" role="alert">{error || status?.message}</p>}
    </form>
    {connected && status ? <div className="fill-body fill-split explorer-split">
      <ModelTree key={`${status.host}:${status.port}`} selected={node} select={picked => { setNode(picked); setView('node'); }}
        fail={setError} />
      <div className="card fill-panel explorer-right">
        <TabBar label={EXPLORER_COPY.views} value={view} change={setView} tabs={[
          ['node', node ? node.split('/').pop() || node : EXPLORER_COPY.nodeTab],
          ['watch', `${EXPLORER_COPY.watchTitle} (${status.watch.length})`]]} />
        {/* Both stay mounted so opened objects and the watch scroll survive a tab switch. */}
        <div className="panel-body explorer-tab" hidden={view !== 'node'}>
          {node ? <NodeDetail reference={node} host={status.host} watched={watched} addWatch={addWatch} fail={setError} />
            : <div className="explorer-detail panel-empty"><div className="guide-empty"><p>{EXPLORER_COPY.pickNode}</p></div></div>}
        </div>
        <div className="panel-body explorer-tab" hidden={view !== 'watch'}><WatchList status={status} run={run} /></div>
      </div>
    </div> : <div className="card fill-body panel-empty"><div className="guide-empty">
      <span className="guide-empty-icon"><Telescope size={24} /></span>
      <h3>{EXPLORER_COPY.notConnected}</h3><p>{EXPLORER_COPY.intro}</p></div></div>}
  </>;
}
