import { Radar, Square } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../../api/client';
import type { Adapter } from '../../../api/network';
import { memoryApi, SCAN_STATUS, scanApi, type ScanPreview, type ScanStatus } from '../../../api/tools';
import { liveFeed } from '../../../api/polls';
import { FIND_COPY } from '../../../copy/tools';
import { HelpTip } from '../../../ui/HelpTip';
import { InfoTip } from '../../../ui/InfoTip';
import { FindResults } from './FindResults';
import { rangeChoices, type RangeChoice } from './ranges';
import '../explorer/explorer.css';
import './find.css';

/** Find IEDs: one toolbar row (adapter, range, port, Find, preview), a thin progress bar, and the
 *  results filling the rest of the window (ADR-0064). */
export function FindTool({ openExplorer, known }: { openExplorer: (host: string, port: number) => void; known: string[] }) {
  const [choices, setChoices] = useState<RangeChoice[]>([]);
  const [choice, setChoice] = useState('');
  const [range, setRange] = useState('');
  const [port, setPort] = useState('102');
  const [preview, setPreview] = useState<ScanPreview | null>(null);
  const [scan, setScan] = useState<ScanStatus | null>(null);
  const [error, setError] = useState('');
  useEffect(() => {
    Promise.all([api<{ adapters: Adapter[] }>('/network/adapters'), memoryApi.get().catch(() => null)]).then(([value, memory]) => {
      const list = rangeChoices(value.adapters, FIND_COPY.thisPc);
      setChoices(list);
      const remembered = list.find(item => item.key === memory?.find.choice);  // the user's last choice
      setChoice((remembered ?? list[0]).key);
      setRange(remembered && memory?.find.range ? memory.find.range : (remembered ?? list[0]).range);
      if (remembered && memory?.find.port) setPort(String(memory.find.port));
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
    if (!scanning) return undefined;
    return liveFeed.subscribe<ScanStatus>(SCAN_STATUS, 300, setScan);
  }, [scanning]);
  const start = () => scanApi.start(range, Number(port)).then(setScan).catch(reason => setError(reason.message));
  const pick = (key: string) => {
    setChoice(key);
    memoryApi.rememberFind(key).catch(() => {});
    const found = choices.find(item => item.key === key);
    if (found) setRange(found.range);
  };
  const percent = scan && scan.total ? Math.round((scan.probed / scan.total) * 100) : 0;
  const help = `${FIND_COPY.intro} ${FIND_COPY.rangeHint}${choice === 'this-pc' ? ` ${FIND_COPY.loopbackHint}` : ''}`;
  return <>
    <div className="card find-bar">
      <div className="find-form">
        <label><InfoTip term="adapter" label={FIND_COPY.adapter} />
          <select value={choice} onChange={event => pick(event.target.value)} disabled={scanning}>
            {choices.map(item => <option key={item.key} value={item.key}>{item.label}</option>)}</select></label>
        <label><InfoTip term="subnet" label={FIND_COPY.range} />
          <input value={range} onChange={event => setRange(event.target.value)} disabled={scanning} spellCheck={false} /></label>
        <label className="find-port">{FIND_COPY.port}
          <input value={port} onChange={event => setPort(event.target.value.replace(/\D/g, ''))} disabled={scanning} /></label>
        {scanning
          ? <button onClick={() => scanApi.stop().then(setScan)}><Square size={16} />{FIND_COPY.stop}</button>
          : <button className="primary" disabled={!preview} onClick={start}><Radar size={16} />
            {scan && scan.state !== 'idle' ? FIND_COPY.again : FIND_COPY.start}</button>}
        {preview && <span className="find-preview" title={FIND_COPY.willCheck(preview.count, preview.first, preview.last, preview.port)}>
          {FIND_COPY.addresses(preview.count)}</span>}
        <HelpTip text={help} label="How does Find IEDs work?" />
      </div>
      {error && <p className="explorer-error find-error" role="alert">{error}</p>}
      <div className={`find-progress${scanning ? '' : ' idle'}`} role="progressbar" aria-label="Scan progress"
        aria-valuenow={scanning ? percent : 0} aria-valuemin={0} aria-valuemax={100}>
        <span style={{ width: `${scanning ? percent : 0}%` }} /></div>
    </div>
    <FindResults scan={scan} percent={percent} known={known} openExplorer={openExplorer} />
  </>;
}
