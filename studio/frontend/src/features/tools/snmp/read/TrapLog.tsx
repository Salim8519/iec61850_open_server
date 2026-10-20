import { BellRing, CircleStop, Trash2 } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { liveFeed } from '../../../../api/polls';
import { snmpApi, type SnmpConfig, type SnmpStatus, type SnmpUser, type TrapEntry } from '../../../../api/snmp';
import { TRAP_RECEIVER_COPY as COPY } from '../../../../copy/snmpRead';
import { Disclosure } from '../../../../ui/Disclosure';
import { HelpTip } from '../../../../ui/HelpTip';
import { ValueText } from '../simulate/ValueEditor';
import { UsersEditor } from '../simulate/UsersEditor';

const KEEP = 500;

/** The trap receiver: where it listens, which v3 users it accepts, and every trap or inform received. */
export function TrapLog({ status, config, refresh, fail }: {
  status: SnmpStatus; config: SnmpConfig; refresh: () => void; fail: (message: string) => void;
}) {
  const [ip, setIp] = useState(config.reader.trap_ip);
  const [port, setPort] = useState(config.reader.trap_port);
  const [users, setUsers] = useState<SnmpUser[]>(config.reader.trap_users);
  const [entries, setEntries] = useState<TrapEntry[]>([]);
  const [busy, setBusy] = useState(false);
  const last = useRef(0);
  useEffect(() => liveFeed.subscribeTask('snmp-traps', 1000,
    () => snmpApi.traps(last.current).then(page => page.entries.length ? page : null), page => {
      const fresh = page.entries.filter(entry => entry.seq > last.current);
      last.current = Math.max(last.current, page.last);
      if (fresh.length) setEntries(current => [...fresh.reverse(), ...current].slice(0, KEEP));
    }), []);
  const act = (task: () => Promise<unknown>) => {
    setBusy(true);
    fail('');
    task().catch(reason => fail(reason.message)).finally(() => { setBusy(false); refresh(); });
  };
  const listening = status.traps.listening;
  return <>
    <div className="panel-toolbar snmp-trap-bar">
      <label className="row">{COPY.where}<select value={ip} disabled={listening} onChange={event => setIp(event.target.value)}>
        <option value="127.0.0.1">{COPY.thisPc}</option><option value="0.0.0.0">{COPY.allCards}</option></select></label>
      <label className="row">{COPY.port}<input className="gateway-number" inputMode="numeric" value={port} disabled={listening}
        onChange={event => setPort(Number(event.target.value) || 0)} /></label>
      {ip === '0.0.0.0' && <HelpTip text={COPY.allCardsHint} />}
      {listening
        ? <button disabled={busy} onClick={() => act(snmpApi.stopListening)}><CircleStop size={16} />{COPY.stop}</button>
        : <button className="primary" disabled={busy} onClick={() => act(() => snmpApi.listen(ip, port, users))}>
          <BellRing size={16} />{COPY.listen}</button>}
      <span className={`small ${listening ? 'snmp-live' : 'muted'}`}>{status.traps.message}</span>
      <span className="gateway-spacer" />
      <button className="ghost" onClick={() => setEntries([])}><Trash2 size={14} />Clear</button>
    </div>
    {!listening && <div className="guide-section panel-messages"><Disclosure title="v3 users">
      <p className="small muted" style={{ margin: 0 }}>{COPY.usersHint}</p>
      <UsersEditor users={users} change={setUsers} title="" /></Disclosure></div>}
    <div className="fill-scroll snmp-trap-list" role="log">
      {entries.length === 0 && <p className="muted watch-empty">{COPY.empty}</p>}
      {entries.map(entry => <article key={entry.seq} className={`snmp-trap${entry.problem ? ' snmp-refused' : ''}`}>
        <header className="row">
          <strong>{entry.notification_name || entry.notification || COPY.problem}</strong>
          <span className="small muted">{entry.version} {entry.kind === 'inform' ? COPY.inform : COPY.trap}
            {' · '}{COPY.from} {entry.from}{entry.user ? ` · user ${entry.user}` : ''}
            {entry.community ? ` · community ${entry.community}` : ''}</span>
          <span className="gateway-spacer" />
          <span className="small muted">{new Date(entry.at * 1000).toLocaleTimeString()}</span>
        </header>
        {entry.problem && <p className="small explorer-error">{entry.problem}</p>}
        {entry.varbinds.length > 0 && <ul className="snmp-trap-values">{entry.varbinds.map(row =>
          <li key={row.oid}><span className="small">{row.name}</span> = <ValueText row={row} /></li>)}</ul>}
      </article>)}
    </div>
  </>;
}
