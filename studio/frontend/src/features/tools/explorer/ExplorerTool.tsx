import { Loader2, Plug, Unplug } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { explorerApi, type Attribute, type ExplorerStatus } from '../../../api/tools';
import { EXPLORER_COPY } from '../../../copy/tools';
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
  connected: { label: EXPLORER_COPY.readOnly, tone: 'good', hint: 'Explorer only reads from this device.' },
  failed: { label: 'Could not connect', tone: 'problem', hint: 'See the message below.' },
  lost: { label: 'Connection lost', tone: 'problem', hint: 'The device closed the connection. Press Connect to try again.' },
};

/** IED Explorer: connect by address, browse one level at a time, and watch chosen values. */
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
    <div className="card">
      <form className="guide-section explorer-connect" onSubmit={event => { event.preventDefault(); connect(host, Number(port)); }}>
        <p className="muted" style={{ margin: 0 }}>{EXPLORER_COPY.intro}</p>
        <div className="row explorer-connect-row">
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
        </div>
        {(error || (status?.state === 'lost' && status.message)) &&
          <p className="explorer-error" role="alert">{error || status?.message}</p>}
      </form>
    </div>
    {connected && status && <>
      <div className="explorer-layout">
        <ModelTree key={`${status.host}:${status.port}`} selected={node} select={setNode} fail={setError} />
        {node ? <NodeDetail reference={node} watched={watched} addWatch={addWatch} fail={setError} />
          : <div className="card explorer-detail"><div className="guide-empty"><p>{EXPLORER_COPY.pickNode}</p></div></div>}
      </div>
      <WatchList status={status} run={run} />
    </>}
  </>;
}
