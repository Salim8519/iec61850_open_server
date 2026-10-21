import { useEffect, useMemo, useState } from 'react';
import { MASTER_COMMANDS, type MasterCommandEntry } from '../../../../api/master';
import { liveFeed } from '../../../../api/polls';
import { GATEWAY_COPY } from '../../../../copy/gateway';
import { COMMAND_NAMES, MASTER_COPY } from '../../../../copy/master';
import { kindOfMasterCommand } from '../../../../points/pointKind';
import { useListFilter } from '../../../../state/useListFilter';
import { FilterBar } from '../../../../ui/FilterBar';
import { HelpTip } from '../../../../ui/HelpTip';
import { KindBadge, kindRow } from '../../../../ui/KindBadge';
import { ValueText } from '../../explorer/ValueText';
import { kindCounts } from '../kindCounts';
import { COMMAND_FILTER, masterCommandMatches } from '../logFilters';

const TONE = { sent: 'pending', confirmed: 'ok', refused: 'warning' } as const;

/** Commands sent from this page and how the device answered each step. The badge is the command's type. */
export function MasterCommandLog() {
  const [entries, setEntries] = useState<MasterCommandEntry[]>([]);
  const filter = useListFilter('master-commands', COMMAND_FILTER);
  useEffect(() => liveFeed.subscribe<{ entries: MasterCommandEntry[] }>(MASTER_COMMANDS, 1000,
    page => setEntries(page.entries)), []);
  const shown = useMemo(() => [...entries].reverse().filter(entry => masterCommandMatches(entry, filter.values)),
    [entries, filter.values]);
  const counts = useMemo(() => kindCounts(entries, entry => kindOfMasterCommand(entry.kind)), [entries]);
  return <>
    <FilterBar label="Filter commands" spec={COMMAND_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={entries.length} counts={counts} extra={<HelpTip text={MASTER_COPY.commandsHint} label="What is this?" />} />
    {entries.length === 0 ? <p className="muted watch-empty">{MASTER_COPY.noCommands}</p> :
      <div className="table-scroll"><table><thead><tr><th>Time</th><th>Type</th><th>Command</th><th>Address</th><th>Value</th>
        <th>From</th><th>Device answer</th></tr></thead>
        <tbody>{shown.map(entry => {
          const kind = kindOfMasterCommand(entry.kind);
          return <tr key={entry.seq} {...kindRow(kind)}>
            <td className="small muted">{new Date(entry.at * 1000).toLocaleTimeString()}</td>
            <td><KindBadge kind={kind} raw={COMMAND_NAMES[entry.kind] ?? entry.kind} /></td>
            <td>{COMMAND_NAMES[entry.kind] ?? entry.kind}{entry.select ? ' · select before operate' : ''}</td>
            <td className="mono">{entry.address ?? '—'}</td>
            <td className="mono">{entry.kind === 'double' && entry.value !== null ? MASTER_COPY.doubleValues[entry.value] ?? entry.value
              : entry.value ?? '—'}</td>
            <td className="small">{entry.who}</td>
            <td><ValueText text={`${MASTER_COPY.states_command[entry.state]}${entry.text ? ` · ${entry.text}` : ''}`}
              status={TONE[entry.state]} /></td>
          </tr>;
        })}</tbody></table>
        {shown.length === 0 && <p className="muted watch-empty">{GATEWAY_COPY.emptyFilter}</p>}</div>}
  </>;
}
