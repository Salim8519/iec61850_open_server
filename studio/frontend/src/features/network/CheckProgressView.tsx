import { Check, CircleAlert, Info, Loader2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import type { CheckProgress } from '../../api/network';
import { liveFeed } from '../../api/polls';
import { StepList } from '../../ui/StepList';
import { checkSteps, ROW_STATE } from './checkSteps';
import './checkProgress.css';

const ICON = { checking: Loader2, free: Check, in_use: CircleAlert, not_checked: Info };

/** While Network setup checks the addresses (change 0150): the step list, then each row turning
 *  free or taken as its answer arrives. Only shows what the engine reports; it decides nothing. */
export function CheckProgressView() {
  const [progress, setProgress] = useState<CheckProgress | null>(null);
  useEffect(() => liveFeed.subscribe<CheckProgress>('/network/plan/progress', 300,
    // A finished earlier check is not this one; keep the first steps until this one reports.
    answer => { if (answer.running) setProgress(answer); }, () => {}), []);
  const rows = progress?.running ? progress.rows : [];
  return <div className="check-progress">
    <StepList title="Checking the addresses" steps={checkSteps(progress)} />
    {rows.length > 0 && <div className="card"><table className="address-table">
      <thead><tr><th>Device</th><th>Address</th><th>Check</th></tr></thead>
      <tbody>{rows.map(row => {
        const Icon = ICON[row.state];
        return <tr key={row.id}>
          <td><strong>{row.name}</strong></td>
          <td>{row.address || '—'}</td>
          <td><span className={`check-state check-${row.state}`}>
            <Icon size={14} aria-hidden="true" className={row.state === 'checking' ? 'ui-spin' : undefined} />
            {ROW_STATE[row.state]}</span></td>
        </tr>;
      })}</tbody></table></div>}
  </div>;
}
