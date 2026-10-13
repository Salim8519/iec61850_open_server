import { ShieldCheck } from 'lucide-react';
import type { ApplyResult, PlanPreview } from '../../api/network';
import { Hint } from '../../ui/Hint';
import { InfoTip } from '../../ui/InfoTip';
import { StatusPill } from '../../ui/StatusPill';
import { UI_STATUS } from '../../copy/status';

export function ReviewStep({ preview }: { preview: PlanPreview }) {
  return <div className="stack">
    <h2 style={{ margin: 0 }}>Review the changes on “{preview.adapter.name}”</h2>
    <Hint id="network.review" />
    <div className="card"><table>
      <thead><tr><th>What will happen</th><th>Kind</th><th>Result</th></tr></thead>
      <tbody>{preview.items.map((item, index) => <tr key={index}>
        <td>{item.description}</td>
        <td>{item.kind === 'alias' ? <InfoTip term="alias" /> : <InfoTip term="firewall" />}</td>
        <td><StatusPill status={UI_STATUS[`item_${item.action}`]} /></td>
      </tr>)}</tbody></table></div>
    {preview.needs_admin && <div className="row" style={{ alignItems: 'flex-start' }}>
      <ShieldCheck size={18} style={{ color: 'var(--primary)', flexShrink: 0 }} />
      <div><strong><InfoTip term="uac" label="Windows will ask for permission" /></strong>
        <Hint id="network.review.uac" compact /></div></div>}
    <Hint id="network.review.reboot" />
  </div>;
}

export function ResultStep({ result }: { result: ApplyResult }) {
  return <div className="stack">
    <h2 style={{ margin: 0 }}>{result.applied ? 'The lab network is ready' : 'Some changes did not work'}</h2>
    <p>{result.message}</p>
    {!!result.results.length && <div className="card"><table><tbody>{result.results.map((entry, index) => <tr key={index}>
      <td>{entry.description}</td>
      <td><StatusPill status={entry.ok ? UI_STATUS.result_done
        : { ...UI_STATUS.result_failed, hint: entry.message || UI_STATUS.result_failed.hint }} showHint={!entry.ok} /></td>
    </tr>)}</tbody></table></div>}
    {result.applied && <Hint id="network.done" />}
  </div>;
}
