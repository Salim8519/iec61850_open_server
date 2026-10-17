import { Loader2, Radar, Search, Square, Telescope } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../../api/client';
import type { Adapter } from '../../../api/network';
import { scanApi, type ScanPreview, type ScanStatus } from '../../../api/tools';
import { FIND_COPY } from '../../../copy/tools';
import { InfoTip } from '../../../ui/InfoTip';
import { StatusPill } from '../../../ui/StatusPill';
import { rangeChoices, type RangeChoice } from './ranges';
import '../explorer/explorer.css';
import './find.css';

/** Find IEDs: choose a range, see exactly what will be checked, scan, and open results in Explorer. */
export function FindTool({ openExplorer }: { openExplorer: (host: string, port: number) => void }) {
  const [choices, setChoices] = useState<RangeChoice[]>([]);
  const [choice, setChoice] = useState('');
  const [range, setRange] = useState('');
  const [port, setPort] = useState('102');
  const [preview, setPreview] = useState<ScanPreview | null>(null);
  const [scan, setScan] = useState<ScanStatus | null>(null);
  const [error, setError] = useState('');
  useEffect(() => {
    api<{ adapters: Adapter[] }>('/network/adapters').then(value => {
      const list = rangeChoices(value.adapters, FIND_COPY.thisPc);
      setChoices(list);
      setChoice(list[0].key); setRange(list[0].range);
    }).catch(reason => setError(reason.message));
    scanApi.status().then(setScan).catch(() => {});
  }, []);
  useEffect(() => {
    if (!range.trim()) { setPreview(null); return; }
    const timer = setTimeout(() => scanApi.preview(range, Number(port) || 0)
      .then(value => { setPreview(value); setError(''); })
      .catch(reason => { setPreview(null); setError(reason.message); }), 250);
    return () => clearTimeout(timer);
  }, [range, port]);
  const scanning = scan?.state === 'scanning';
  useEffect(() => {
    if (!scanning) return;
    const timer = setInterval(() => scanApi.status().then(setScan).catch(() => {}), 300);
    return () => clearInterval(timer);
  }, [scanning]);
  const start = () => scanApi.start(range, Number(port)).then(setScan).catch(reason => setError(reason.message));
  const pick = (key: string) => {
    setChoice(key);
    const found = choices.find(item => item.key === key);
    if (found) setRange(found.range);
  };
  const percent = scan && scan.total ? Math.round((scan.probed / scan.total) * 100) : 0;
  return <>
    <div className="card"><div className="guide-section">
      <p className="muted" style={{ margin: 0 }}>{FIND_COPY.intro}</p>
      <div className="row find-form">
        <label><InfoTip term="adapter" label={FIND_COPY.adapter} />
          <select value={choice} onChange={event => pick(event.target.value)} disabled={scanning}>
            {choices.map(item => <option key={item.key} value={item.key}>{item.label}</option>)}</select></label>
        <label><InfoTip term="subnet" label={FIND_COPY.range} />
          <input value={range} onChange={event => setRange(event.target.value)} disabled={scanning} spellCheck={false} /></label>
        <label className="explorer-port">{FIND_COPY.port}
          <input value={port} onChange={event => setPort(event.target.value.replace(/\D/g, ''))} disabled={scanning} /></label>
        {scanning
          ? <button onClick={() => scanApi.stop().then(setScan)}><Square size={16} />{FIND_COPY.stop}</button>
          : <button className="primary" disabled={!preview} onClick={start}><Radar size={16} />
            {scan && scan.state !== 'idle' ? FIND_COPY.again : FIND_COPY.start}</button>}
      </div>
      <p className="small muted" style={{ margin: 0 }}>{FIND_COPY.rangeHint}</p>
      {choice === 'this-pc' && <p className="small muted" style={{ margin: 0 }}>{FIND_COPY.loopbackHint}</p>}
      {preview && !scanning && <p className="find-preview"><Search size={15} />
        {FIND_COPY.willCheck(preview.count, preview.first, preview.last, preview.port)}</p>}
      {error && <p className="explorer-error" role="alert">{error}</p>}
    </div></div>
    {scan && scan.state !== 'idle' && <div className="card">
      <div className="card-heading"><h3>Results</h3>
        <StatusPill status={scanning ? { label: `Scanning ${percent}%`, tone: 'waiting', hint: scan.message }
          : scan.state === 'failed' ? { label: 'Scan failed', tone: 'problem', hint: scan.message }
          : { label: `${scan.found.length} found`, tone: scan.found.length ? 'good' : 'off', hint: scan.message }} /></div>
      {scanning && <div className="find-progress" role="progressbar" aria-valuenow={percent} aria-valuemin={0} aria-valuemax={100}>
        <span style={{ width: `${percent}%` }} /></div>}
      <div className="guide-section"><p className="small muted" style={{ margin: 0 }} aria-live="polite">{scan.message}
        {!scanning && scan.seconds ? ` (${scan.seconds} s)` : ''}</p></div>
      {scan.found.length > 0 ? <table><thead><tr><th>Address</th><th>Vendor</th><th>Model</th><th>Firmware</th>
        <th>Logical devices</th><th>Answer</th><th aria-label="Open" /></tr></thead>
        <tbody>{scan.found.map(device => <tr key={device.ip}>
          <td className="mono"><strong>{device.ip}</strong>{device.port !== 102 ? `:${device.port}` : ''}</td>
          <td>{device.identified === null ? <span className="muted"><Loader2 size={13} className="spin" /> Asking…</span>
            : device.vendor || <span className="muted">—</span>}</td>
          <td>{device.model || <span className="muted">—</span>}</td><td>{device.revision || <span className="muted">—</span>}</td>
          <td className="small">{device.devices.length ? device.devices.join(', ') : <span className="muted">{device.reason}</span>}</td>
          <td className="small muted">{device.tcp_ms < 1 ? '<1' : Math.round(device.tcp_ms)} ms</td>
          <td><button disabled={device.identified !== true} onClick={() => openExplorer(device.ip, device.port)}>
            <Telescope size={15} />{FIND_COPY.open}</button></td></tr>)}</tbody></table>
        : !scanning && <p className="muted watch-empty">{FIND_COPY.none}</p>}
      {scan.found.length > 0 && !scanning && <div className="guide-section"><p className="small muted" style={{ margin: 0 }}>
        {FIND_COPY.addHint}</p></div>}
    </div>}
  </>;
}
