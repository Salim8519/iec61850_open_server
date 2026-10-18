import { Loader2, Plug, Telescope, Unplug } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { explorerApi, type Attribute, type ExplorerStatus } from '../../../api/tools';
import { EXPLORER_COPY } from '../../../copy/tools';
import { HelpTip } from '../../../ui/HelpTip';
import { InfoTip } from '../../../ui/InfoTip';
import { StatusPill } from '../../../ui/StatusPill';
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

/** IED Explorer: connect by address, browse one level at a time, and watch chosen values. A toolbar row,
 *  then the model tree on the left and the node with the watch list pinned below it (ADR-0064). */
export function ExplorerTool({ target }: { target?: { host: string; port: number } }) {
  const [status, setStatus] = useState<ExplorerStatus | null>(null);
  const [host, setHost] = useState(target?.host ?? '');
  const [port, setPort] = useState(String(target?.port ?? 102));
  const [node, setNode] = useState('');
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
  useEffect(() => {
    explorerApi.status().then(value => {
      setStatus(value);
      if (target && !(value.state === 'connected' && value.host === target.host && value.port === target.port))
        connect(target.host, target.port);
      else if (value.host && !target) { setHost(value.host); setPort(String(value.port)); }
    }).catch(reason => setError(reason.message));
  }, [target, connect]);
  const connected = status?.state === 'connected';
  useEffect(() => {
    if (!connected) return;
    const timer = setInterval(() => explorerApi.status().then(setStatus).catch(() => {}),
      status?.polling ? Math.max(500, Math.min(status.interval_ms, 2000)) : 4000);
    return () => clearInterval(timer);
  }, [connected, status?.polling, status?.interval_ms]);
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
      <ModelTree key={`${status.host}:${status.port}`} selected={node} select={setNode} fail={setError} />
      <div className="fill-column">
        {node ? <NodeDetail reference={node} host={status.host} watched={watched} addWatch={addWatch} fail={setError} />
          : <div className="card explorer-detail fill-grow panel-empty"><div className="guide-empty"><p>{EXPLORER_COPY.pickNode}</p></div></div>}
        <WatchList status={status} run={run} />
      </div>
    </div> : <div className="card fill-body panel-empty"><div className="guide-empty">
      <span className="guide-empty-icon"><Telescope size={24} /></span>
      <h3>{EXPLORER_COPY.notConnected}</h3><p>{EXPLORER_COPY.intro}</p></div></div>}
  </>;
}
