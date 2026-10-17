import { ArrowDownLeft, ArrowUpRight, Info, Pause, Play, Search, Trash2 } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { gatewayApi, type TrafficEntry } from '../../../api/gateway';
import { GATEWAY_COPY } from '../../../copy/gateway';

const KEEP = 1000;
const DIRECTION = { rx: { icon: ArrowDownLeft, word: 'from SCADA' }, tx: { icon: ArrowUpRight, word: 'to SCADA' },
  info: { icon: Info, word: 'note' } } as const;

/** Decoded traffic, newest at the bottom; it follows new lines unless paused. */
export function TrafficLog({ running }: { running: boolean }) {
  const [lines, setLines] = useState<TrafficEntry[]>([]);
  const [paused, setPaused] = useState(false);
  const [filter, setFilter] = useState('');
  const last = useRef(0);
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (paused) return;
    const load = () => gatewayApi.traffic(last.current).then(page => {
      if (!page.entries.length) return;
      last.current = page.last;
      setLines(current => [...current, ...page.entries].slice(-KEEP));
    }).catch(() => {});
    load();
    const timer = setInterval(load, running ? 500 : 2000);
    return () => clearInterval(timer);
  }, [paused, running]);
  useEffect(() => {
    if (!paused && box.current) box.current.scrollTop = box.current.scrollHeight;
  }, [lines, paused]);
  const needle = filter.trim().toLowerCase();
  const shown = needle ? lines.filter(line => `${line.peer} ${line.text}`.toLowerCase().includes(needle)) : lines;
  return <>
    <div className="guide-section row gateway-toolbar">
      <span className="small muted">{GATEWAY_COPY.trafficHint}</span>
      <span className="gateway-spacer" />
      <label className="search-field"><Search size={16} /><input value={filter} placeholder={GATEWAY_COPY.filter}
        aria-label={GATEWAY_COPY.filter} onChange={event => setFilter(event.target.value)} /></label>
      <button onClick={() => setPaused(!paused)}>{paused ? <><Play size={16} />{GATEWAY_COPY.resume}</>
        : <><Pause size={16} />{GATEWAY_COPY.pause}</>}</button>
      <button className="ghost" onClick={() => setLines([])}><Trash2 size={16} />{GATEWAY_COPY.clear}</button>
    </div>
    <div className="gateway-traffic" ref={box} role="log" aria-live={paused ? 'off' : 'polite'}>
      {shown.length === 0 && <p className="muted">No messages yet.</p>}
      {shown.map(line => {
        const direction = DIRECTION[line.dir as keyof typeof DIRECTION] ?? DIRECTION.info;
        const Icon = direction.icon;
        return <div key={line.seq} className={`traffic-line dir-${line.dir}`}>
          <span className="traffic-time">{new Date(line.at * 1000).toLocaleTimeString()}</span>
          <Icon size={14} aria-label={direction.word} />
          {line.peer && <span className="traffic-peer">{line.peer}</span>}
          <span className="traffic-text">{line.text}</span>
        </div>;
      })}
    </div>
  </>;
}
