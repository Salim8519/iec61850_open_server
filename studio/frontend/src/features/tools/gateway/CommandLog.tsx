import { useEffect, useState } from 'react';
import { gatewayApi, type CommandEntry } from '../../../api/gateway';
import { GATEWAY_COPY, PROTOCOL_NAMES } from '../../../copy/gateway';
import type { Protocol } from '../../../api/gateway';
import { useListFilter } from '../../../state/useListFilter';
import { FilterBar } from '../../../ui/FilterBar';
import { HelpTip } from '../../../ui/HelpTip';
import { ValueText } from '../explorer/ValueText';
import { COMMAND_FILTER, gatewayCommandMatches } from './logFilters';

const status = (text: string) => !text ? 'pending' as const
  : text.startsWith('operated') || text.includes('is now') ? 'ok' as const
  : text.startsWith('refused') ? 'warning' as const : 'problem' as const;

/** Commands from SCADA or from this page: what was asked, what the IED did, what was seen afterwards. */
export function CommandLog() {
  const [entries, setEntries] = useState<CommandEntry[]>([]);
  const filter = useListFilter('commands', COMMAND_FILTER);
  useEffect(() => {
    const load = () => gatewayApi.commands().then(page => setEntries(page.entries)).catch(() => {});
    load();
    const timer = setInterval(load, 1000);
    return () => clearInterval(timer);
  }, []);
  const shown = [...entries].reverse().filter(entry => gatewayCommandMatches(entry, filter.values));
  return <>
    <FilterBar label="Filter commands" spec={COMMAND_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={entries.length} extra={<HelpTip text={GATEWAY_COPY.commandsHint} label="What is this?" />} />
    {entries.length === 0 ? <p className="muted watch-empty">{GATEWAY_COPY.noCommands}</p> :
      <div className="table-scroll"><table><thead><tr><th>Time</th><th>Command</th><th>From</th><th>Applied</th>
        <th>Observed</th></tr></thead>
        <tbody>{shown.map(entry => <tr key={entry.seq}>
          <td className="small muted">{new Date(entry.at * 1000).toLocaleTimeString()}</td>
          <td className="mono reference-cell" title={entry.label}>{entry.label}</td>
          <td className="small">{entry.protocol === 'manual' ? GATEWAY_COPY.fromPage
            : `${PROTOCOL_NAMES[entry.protocol as Protocol] ?? entry.protocol} · ${entry.peer}`}</td>
          <td><ValueText text={entry.applied || 'waiting…'} status={status(entry.applied)} /></td>
          <td><ValueText text={entry.observed || 'waiting…'} status={entry.observed.includes('did not change')
            ? 'warning' : status(entry.observed)} /></td>
        </tr>)}</tbody></table>
        {shown.length === 0 && <p className="muted watch-empty">{GATEWAY_COPY.emptyFilter}</p>}</div>}
  </>;
}
