import { Trash2 } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { liveFeed } from '../../../../api/polls';
import { snmpApi, type RequestEntry } from '../../../../api/snmp';
import { LOG_COPY } from '../../../../copy/snmp';

const KEEP = 1000;

/** The requests this device answered (or ignored, with the reason), newest at the bottom. */
export function RequestLog({ device }: { device: string }) {
  const [lines, setLines] = useState<RequestEntry[]>([]);
  const last = useRef(0);
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => liveFeed.subscribeTask('snmp-requests', 1000,
    () => snmpApi.requests(last.current).then(page => page.entries.length ? page : null), page => {
      const fresh = page.entries.filter(entry => entry.seq > last.current);
      last.current = Math.max(last.current, page.last);
      if (fresh.length) setLines(current => [...current, ...fresh].slice(-KEEP));
    }), []);
  const shown = useMemo(() => lines.filter(line => line.device === device), [lines, device]);
  useEffect(() => {
    if (box.current) box.current.scrollTop = box.current.scrollHeight;
  }, [shown.length]);
  return <>
    <div className="panel-toolbar">
      <span className="small muted">{shown.length} {shown.length === 1 ? 'request' : 'requests'}</span>
      <span className="gateway-spacer" />
      <button className="ghost" onClick={() => setLines(current => current.filter(line => line.device !== device))}>
        <Trash2 size={14} />Clear</button>
    </div>
    <div className="gateway-traffic" ref={box} role="log">
      {shown.length === 0 && <p className="muted">{LOG_COPY.empty}</p>}
      {shown.map(line => <div key={line.seq} className={`traffic-line${line.result.startsWith('ok') || line.result === 'sent'
        || line.result === 'acknowledged' ? '' : ' snmp-refused'}`}>
        <span className="traffic-time">{new Date(line.at * 1000).toLocaleTimeString()}</span>
        <span>{line.peer}</span>
        <strong>{line.version} {line.op}</strong>
        <span title={line.oid}>{line.name || line.oid}{line.count > 1 ? LOG_COPY.andMore(line.count - 1) : ''}</span>
        <span>{line.result}{line.held > 0 ? LOG_COPY.held(line.held) : ''}</span>
      </div>)}
    </div>
  </>;
}
