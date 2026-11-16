import { AlertTriangle, CheckCircle2, HelpCircle } from 'lucide-react';
import { useEffect, useState } from 'react';
import { api } from '../../api/client';
import type { CommandCheck } from '../../api/network';
import { hint } from '../../copy/hints';
import { HelpTip } from '../../ui/HelpTip';

const MARK = {
  differs: { Icon: AlertTriangle, color: 'var(--warning)', word: 'Different' },
  unreadable: { Icon: HelpCircle, color: 'var(--muted)', word: 'Not readable' },
};

/** How the real device takes each command of the CID (ADR-0138). The engine reads and classes it;
 *  this card only shows the answer. Read again when the session connects. */
export function DeviceCommandCheck({ iedId, live }: { iedId: string; live: boolean }) {
  const [check, setCheck] = useState<CommandCheck | null>(null);
  useEffect(() => {
    setCheck(null);
    if (!live) return;
    let current = true;
    api<CommandCheck>(`/live/${iedId}/controls`).then(answer => { if (current) setCheck(answer); }).catch(() => undefined);
    return () => { current = false; };
  }, [iedId, live]);
  if (!check) return null;
  const summary = check.summary;
  const open = (check.rows ?? []).filter(row => row.result !== 'match');
  return <div className="stack" style={{ gap: 8 }}>
    <span className="label-row"><span className="section-label">Commands on the device</span>
      <HelpTip text={hint('live.command_check')} label="What is checked?" /></span>
    {check.error && <p className="small" role="alert" style={{ margin: 0 }}>{check.error}</p>}
    {summary && (summary.total === 0
      ? <span className="small muted">The CID has no commands for this IED.</span>
      : <span className="small row" style={{ alignItems: 'flex-start' }}>
        {open.length ? <AlertTriangle size={15} style={{ flexShrink: 0, color: 'var(--warning)' }} aria-hidden />
          : <CheckCircle2 size={15} style={{ flexShrink: 0, color: 'var(--success)' }} aria-hidden />}
        {summary.total} commands checked: {summary.match} as in the CID, {summary.differs} different, {summary.unreadable} not readable.
      </span>)}
    {!!check.local?.length && <div className="warning-banner" role="status"><span>
      <strong>The device is in Local</strong> ({check.local.join(', ')}). It refuses remote commands until it is switched to Remote.</span></div>}
    {check.limited && <span className="small muted">Only the first 500 commands were checked.</span>}
    {!!open.length && <ul className="stack" style={{ gap: 6, margin: 0, paddingLeft: 0, listStyle: 'none', maxHeight: 220, overflow: 'auto' }}>
      {open.map(row => {
        const mark = MARK[row.result as keyof typeof MARK];
        return <li key={row.ref} className="small row" style={{ alignItems: 'flex-start' }}>
          <mark.Icon size={15} style={{ flexShrink: 0, color: mark.color }} aria-hidden />
          <span><strong>{mark.word}</strong> <code>{row.ref}</code> {row.note}</span></li>;
      })}</ul>}
  </div>;
}
