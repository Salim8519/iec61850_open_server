import { useEffect, useMemo, useState } from 'react';
import { GATEWAY_COMMANDS, type Protocol } from '../../../api/gateway';
import { liveFeed } from '../../../api/polls';
import { GATEWAY_COPY, PROTOCOL_NAMES } from '../../../copy/gateway';
import { useListFilter } from '../../../state/useListFilter';
import { FilterBar } from '../../../ui/FilterBar';
import { HelpTip } from '../../../ui/HelpTip';
import { KindBadge, kindRow } from '../../../ui/KindBadge';
import { ValueText } from '../explorer/ValueText';
import { gatewayCommandKind, gatewayCommandRaw, type LoggedCommand } from './commandKinds';
import { kindCounts } from './kindCounts';
import { COMMAND_FILTER, gatewayCommandMatches } from './logFilters';
import { useCommandKinds } from './useCommandKinds';

const status = (text: string) => !text ? 'pending' as const
  : text.startsWith('operated') || text.includes('is now') ? 'ok' as const
  : text.startsWith('refused') ? 'warning' as const : 'problem' as const;

/** Commands from SCADA or from this page: what was asked, what the IED did, what was seen afterwards.
 *  The badge is the type of the point the command operated (single or double command). */
export function CommandLog({ protocol }: { protocol: Protocol }) {
  const [entries, setEntries] = useState<LoggedCommand[]>([]);
  const filter = useListFilter('commands', COMMAND_FILTER);
  useEffect(() => liveFeed.subscribe<{ entries: LoggedCommand[] }>(GATEWAY_COMMANDS, 1000,
    page => setEntries(page.entries)), []);
  const kinds = useCommandKinds(protocol, entries);
  const shown = useMemo(() => [...entries].reverse().filter(entry =>
    gatewayCommandMatches(entry, filter.values, gatewayCommandKind(entry, kinds))), [entries, filter.values, kinds]);
  const counts = useMemo(() => kindCounts(entries, entry => gatewayCommandKind(entry, kinds)), [entries, kinds]);
  return <>
    <FilterBar label="Filter commands" spec={COMMAND_FILTER} values={filter.values} set={filter.set} clear={filter.clear}
      shown={shown.length} total={entries.length} counts={counts} extra={<HelpTip text={GATEWAY_COPY.commandsHint} label="What is this?" />} />
    {entries.length === 0 ? <p className="muted watch-empty">{GATEWAY_COPY.noCommands}</p> :
      <div className="table-scroll"><table><thead><tr><th>Time</th><th>Type</th><th>Command</th><th>From</th><th>Applied</th>
        <th>Observed</th></tr></thead>
        <tbody>{shown.map(entry => {
          const kind = gatewayCommandKind(entry, kinds);
          return <tr key={entry.seq} {...kindRow(kind)}>
            <td className="small muted">{new Date(entry.at * 1000).toLocaleTimeString()}</td>
            <td><KindBadge kind={kind} raw={gatewayCommandRaw(entry, kinds)} /></td>
            <td className="mono reference-cell" title={entry.label}>{entry.label}</td>
            <td className="small">{entry.protocol === 'manual' ? GATEWAY_COPY.fromPage
              : `${PROTOCOL_NAMES[entry.protocol as Protocol] ?? entry.protocol} · ${entry.peer}`}</td>
            <td><ValueText text={entry.applied || 'waiting…'} status={status(entry.applied)} /></td>
            <td><ValueText text={entry.observed || 'waiting…'} status={entry.observed.includes('did not change')
              ? 'warning' : status(entry.observed)} /></td>
          </tr>;
        })}</tbody></table>
        {shown.length === 0 && <p className="muted watch-empty">{GATEWAY_COPY.emptyFilter}</p>}</div>}
  </>;
}
