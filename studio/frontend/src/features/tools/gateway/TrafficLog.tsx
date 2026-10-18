import { ArrowDownLeft, ArrowUpRight, Info, Pause, Play, Trash2 } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { gatewayApi, type TrafficEntry } from '../../../api/gateway';
import { GATEWAY_COPY } from '../../../copy/gateway';
import { useListFilter } from '../../../state/useListFilter';
import { FilterBar } from '../../../ui/FilterBar';
import { HelpTip } from '../../../ui/HelpTip';
import { TRAFFIC_FILTER, trafficMatches } from './logFilters';

const KEEP = 1000;
const ICONS = { rx: ArrowDownLeft, tx: ArrowUpRight, info: Info } as const;
const SCADA_WORDS = { rx: 'from SCADA', tx: 'to SCADA', info: 'note' };
type Load = (after: number) => Promise<{ entries: TrafficEntry[]; last: number }>;

/** Decoded traffic, newest at the bottom; it follows new lines unless paused. The master view passes its own source. */
export function TrafficLog({ running, load: source = gatewayApi.traffic, words = SCADA_WORDS, hint = GATEWAY_COPY.trafficHint }: {
  running: boolean; load?: Load; words?: Record<'rx' | 'tx' | 'info', string>; hint?: string;
}) {
  const [lines, setLines] = useState<TrafficEntry[]>([]);
  const [paused, setPaused] = useState(false);
  const filter = useListFilter('traffic', TRAFFIC_FILTER);
  const last = useRef(0);
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (paused) return;
    const load = () => source(last.current).then(page => {
      if (!page.entries.length) return;
      last.current = page.last;
      setLines(current => [...current, ...page.entries].slice(-KEEP));
    }).catch(() => {});
    load();
    const timer = setInterval(load, running ? 500 : 2000);
    return () => clearInterval(timer);
  }, [paused, running, source]);
  const shown = lines.filter(line => trafficMatches(line, filter.values));
  useEffect(() => {
    if (!paused && box.current) box.current.scrollTop = box.current.scrollHeight;
  }, [shown.length, paused]);
  return <>
    <FilterBar label="Filter messages" spec={TRAFFIC_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={lines.length} extra={<>
        <button className="chip toggle" aria-pressed={paused} onClick={() => setPaused(!paused)}>
          {paused ? <><Play size={14} />{GATEWAY_COPY.resume}</> : <><Pause size={14} />{GATEWAY_COPY.pause}</>}</button>
        <button className="ghost filter-clear" onClick={() => setLines([])}><Trash2 size={14} />{GATEWAY_COPY.clear}</button>
        <HelpTip text={hint} label="What is this?" /></>} />
    <div className="gateway-traffic" ref={box} role="log" aria-live={paused ? 'off' : 'polite'}>
      {shown.length === 0 && <p className="muted">{lines.length ? GATEWAY_COPY.emptyFilter : 'No messages yet.'}</p>}
      {shown.map(line => {
        const dir = (line.dir in ICONS ? line.dir : 'info') as keyof typeof ICONS;
        const Icon = ICONS[dir];
        return <div key={line.seq} className={`traffic-line dir-${line.dir}`}>
          <span className="traffic-time">{new Date(line.at * 1000).toLocaleTimeString()}</span>
          <Icon size={14} aria-label={words[dir]} />
          {line.peer && <span className="traffic-peer">{line.peer}</span>}
          <span className="traffic-text">{line.text}</span>
        </div>;
      })}
    </div>
  </>;
}
