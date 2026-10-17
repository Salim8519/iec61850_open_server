import { useEffect, useState } from 'react';
import { gatewayApi, type CommandEntry } from '../../../api/gateway';
import { GATEWAY_COPY } from '../../../copy/gateway';
import { ValueText } from '../explorer/ValueText';

const status = (text: string) => !text ? 'pending' as const
  : text.startsWith('operated') || text.includes('is now') ? 'ok' as const
  : text.startsWith('refused') ? 'warning' as const : 'problem' as const;

/** SCADA commands: what was asked, what the simulated IED did, what was seen afterwards. */
export function CommandLog() {
  const [entries, setEntries] = useState<CommandEntry[]>([]);
  useEffect(() => {
    const load = () => gatewayApi.commands().then(page => setEntries(page.entries)).catch(() => {});
    load();
    const timer = setInterval(load, 1000);
    return () => clearInterval(timer);
  }, []);
  return <>
    <div className="guide-section"><p className="small muted" style={{ margin: 0 }}>{GATEWAY_COPY.commandsHint}</p></div>
    {entries.length === 0 ? <p className="muted watch-empty">No command from SCADA yet.</p> :
      <div className="table-scroll"><table><thead><tr><th>Time</th><th>Command</th><th>From</th><th>Applied</th>
        <th>Observed</th></tr></thead>
        <tbody>{[...entries].reverse().map(entry => <tr key={entry.seq}>
          <td className="small muted">{new Date(entry.at * 1000).toLocaleTimeString()}</td>
          <td className="mono reference-cell" title={entry.label}>{entry.label}</td>
          <td className="small">{entry.protocol} · {entry.peer}</td>
          <td><ValueText text={entry.applied || 'waiting…'} status={status(entry.applied)} /></td>
          <td><ValueText text={entry.observed || 'waiting…'} status={entry.observed.includes('did not change')
            ? 'warning' : status(entry.observed)} /></td>
        </tr>)}</tbody></table></div>}
  </>;
}
