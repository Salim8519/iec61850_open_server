import { useEffect, useState } from 'react';
import { masterApi, type MasterCommandEntry } from '../../../../api/master';
import { COMMAND_NAMES, MASTER_COPY } from '../../../../copy/master';
import { ValueText } from '../../explorer/ValueText';

const TONE = { sent: 'pending', confirmed: 'ok', refused: 'warning' } as const;

/** Commands sent from this page and how the device answered each step. */
export function MasterCommandLog() {
  const [entries, setEntries] = useState<MasterCommandEntry[]>([]);
  useEffect(() => {
    const load = () => masterApi.commands().then(page => setEntries(page.entries)).catch(() => {});
    load();
    const timer = setInterval(load, 1000);
    return () => clearInterval(timer);
  }, []);
  return <>
    <div className="guide-section"><p className="small muted" style={{ margin: 0 }}>{MASTER_COPY.commandsHint}</p></div>
    {entries.length === 0 ? <p className="muted watch-empty">{MASTER_COPY.noCommands}</p> :
      <div className="table-scroll"><table><thead><tr><th>Time</th><th>Command</th><th>Address</th><th>Value</th>
        <th>From</th><th>Device answer</th></tr></thead>
        <tbody>{[...entries].reverse().map(entry => <tr key={entry.seq}>
          <td className="small muted">{new Date(entry.at * 1000).toLocaleTimeString()}</td>
          <td>{COMMAND_NAMES[entry.kind] ?? entry.kind}{entry.select ? ' · select before operate' : ''}</td>
          <td className="mono">{entry.address ?? '—'}</td>
          <td className="mono">{entry.kind === 'double' && entry.value !== null ? MASTER_COPY.doubleValues[entry.value] ?? entry.value
            : entry.value ?? '—'}</td>
          <td className="small">{entry.who}</td>
          <td><ValueText text={`${MASTER_COPY.states_command[entry.state]}${entry.text ? ` · ${entry.text}` : ''}`}
            status={TONE[entry.state]} /></td>
        </tr>)}</tbody></table></div>}
  </>;
}
